package normalizer

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/marquesfelip/d365-fo-db-diagram/internal/model"
	"github.com/marquesfelip/d365-fo-db-diagram/payload"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Normalizer struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Normalizer {
	return &Normalizer{db: db}
}

type fieldKey struct {
	TableID uint
	Name    string
}

// Run reads all unprocessed raw_ax_table records, normalizes them and populates
// ax_table, edt, table_field, table_relation, and relation_field.
// Returns the number of raw records processed.
func (n *Normalizer) Run() (int, error) {
	// 1. Fetch unprocessed raw records
	var raws []model.RawAxTable
	if err := n.db.Where("processed = ?", false).Find(&raws).Error; err != nil {
		return 0, fmt.Errorf("fetching raw records: %w", err)
	}

	if len(raws) == 0 {
		return 0, nil
	}

	// 2. Parse payloads and deduplicate by table name (same table can appear
	//    in multiple batches if the ingest is run more than once).
	type tableEntry struct {
		raw     model.RawAxTable
		payload payload.AxTable
	}

	tablesByName := make(map[string]tableEntry, len(raws))
	rawIDs := make([]uint, 0, len(raws))

	for _, raw := range raws {
		var p payload.AxTable
		if err := json.Unmarshal(raw.Payload, &p); err != nil {
			log.Printf("normalizer: skipping raw id=%d (%s): %v", raw.ID, raw.Name, err)
			continue
		}
		if _, exists := tablesByName[p.Name]; !exists {
			tablesByName[p.Name] = tableEntry{raw: raw, payload: p}
		}
		rawIDs = append(rawIDs, raw.ID)
	}

	// 3. Insert ax_table rows (skip conflicts – same name already in DB)
	axTableRows := make([]model.AxTable, 0, len(tablesByName))
	for _, entry := range tablesByName {
		p := entry.payload
		row := model.AxTable{
			Name:               p.Name,
			Model:              entry.raw.Model,
			Layer:              entry.raw.Layer,
			Extends:            nilIfEmpty(p.Extends),
			SaveDataPerCompany: p.SaveDataPerCompany,
			TableGroup:         p.TableGroup,
			TableType:          p.TableType,
			PrimaryIndex:       p.PrimaryIndex,
			ReplacementKey:     p.ReplacementKey,
		}
		axTableRows = append(axTableRows, row)
	}

	if len(axTableRows) > 0 {
		res := n.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).CreateInBatches(&axTableRows, 1000)
		if res.Error != nil {
			return 0, fmt.Errorf("inserting ax_tables: %w", res.Error)
		}
	}

	// 4. Load all ax_tables from DB to build name→id map
	var allAxTables []struct {
		ID   uint   `gorm:"column:id_ax_table"`
		Name string `gorm:"column:name"`
	}
	if err := n.db.Model(&model.AxTable{}).Find(&allAxTables).Error; err != nil {
		return 0, fmt.Errorf("loading ax_tables: %w", err)
	}
	axTableMap := make(map[string]uint, len(allAxTables))
	for _, t := range allAxTables {
		axTableMap[t.Name] = t.ID
	}

	// 5. Resolve fk_extends for the current batch (best-effort; cross-batch
	//    references to tables not yet in the DB will remain NULL).
	for _, entry := range tablesByName {
		extends := entry.payload.Extends
		if extends == "" {
			continue
		}
		if extendsID, ok := axTableMap[extends]; ok {
			n.db.Model(&model.AxTable{}).
				Where("name = ? AND fk_extends IS NULL", entry.payload.Name).
				Update("fk_extends", extendsID)
		}
	}

	// 6. Collect unique EDT names referenced by table fields
	edtNameSet := make(map[string]bool)
	for _, entry := range tablesByName {
		for _, f := range entry.payload.Fields.AxTableField {
			if f.ExtendedDataType != "" {
				edtNameSet[f.ExtendedDataType] = true
			}
		}
	}

	// 7. Insert EDT stubs (name only; full EDT metadata comes from AxEdt files)
	if len(edtNameSet) > 0 {
		edtRows := make([]model.Edt, 0, len(edtNameSet))
		for name := range edtNameSet {
			edtRows = append(edtRows, model.Edt{Name: name})
		}
		res := n.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).CreateInBatches(edtRows, 1000)
		if res.Error != nil {
			return 0, fmt.Errorf("inserting edts: %w", res.Error)
		}
	}

	// 8. Load all EDTs to build name→id map
	var allEdts []struct {
		ID   uint   `gorm:"column:id_edt"`
		Name string `gorm:"column:name"`
	}
	if err := n.db.Model(&model.Edt{}).Find(&allEdts).Error; err != nil {
		return 0, fmt.Errorf("loading edts: %w", err)
	}
	edtMap := make(map[string]uint, len(allEdts))
	for _, e := range allEdts {
		edtMap[e.Name] = e.ID
	}

	// 9. Insert table_field rows
	var fieldRows []model.TableField
	for _, entry := range tablesByName {
		axID, ok := axTableMap[entry.payload.Name]
		if !ok {
			continue
		}
		for _, f := range entry.payload.Fields.AxTableField {
			row := model.TableField{
				FkAxTable: &axID,
				Name:      f.Name,
				Edt:       nilIfEmpty(f.ExtendedDataType),
			}
			if f.ExtendedDataType != "" {
				if edtID, ok := edtMap[f.ExtendedDataType]; ok {
					id := int(edtID)
					row.FkEdt = &id
				}
			}
			fieldRows = append(fieldRows, row)
		}
	}

	if len(fieldRows) > 0 {
		if err := n.db.CreateInBatches(&fieldRows, 1000).Error; err != nil {
			return 0, fmt.Errorf("inserting table_fields: %w", err)
		}
	}

	// 10. Build (ax_table_id, field_name) → field_id map for FK resolution
	fieldMap := make(map[fieldKey]uint, len(fieldRows))
	for _, f := range fieldRows {
		if f.FkAxTable != nil && f.ID > 0 {
			fieldMap[fieldKey{*f.FkAxTable, f.Name}] = f.ID
		}
	}

	// 11. Build and insert table_relation rows
	type relEntry struct {
		relation payload.AxTableRelation
		row      model.TableRelation
	}
	var relEntries []relEntry

	for _, entry := range tablesByName {
		for _, rel := range entry.payload.Relations.AxTableRelation {
			row := model.TableRelation{
				Name:                    rel.Name,
				SourceTable:             entry.payload.Name,
				RelatedTable:            rel.RelatedTable,
				OnDelete:                nilIfEmpty(rel.OnDelete),
				Cardinality:             nilIfEmpty(rel.Cardinality),
				RelatedTableCardinality: rel.RelatedTableCardinality,
				RelationshipType:        rel.RelationshipType,
			}
			if rel.EDTRelation {
				s := "Yes"
				row.EdtRelation = &s
			}
			if sID, ok := axTableMap[entry.payload.Name]; ok {
				row.FkSourceTable = &sID
			}
			if rID, ok := axTableMap[rel.RelatedTable]; ok {
				row.FkRelatedTable = &rID
			}
			relEntries = append(relEntries, relEntry{rel, row})
		}
	}

	if len(relEntries) > 0 {
		rows := make([]model.TableRelation, len(relEntries))
		for i, e := range relEntries {
			rows[i] = e.row
		}
		if err := n.db.CreateInBatches(&rows, 1000).Error; err != nil {
			return 0, fmt.Errorf("inserting table_relations: %w", err)
		}
		for i := range relEntries {
			relEntries[i].row = rows[i]
		}
	}

	// 12. Insert relation_field rows (one per constraint inside each relation)
	var relFieldRows []model.RelationField
	for _, entry := range relEntries {
		relID := entry.row.ID
		if relID == 0 {
			continue
		}
		sourceTableID, _ := axTableMap[entry.row.SourceTable]
		relatedTableID, _ := axTableMap[entry.row.RelatedTable]

		for _, c := range entry.relation.Constraints.AxTableRelationConstraint {
			row := model.RelationField{
				FkTableRelation: &relID,
				Name:            c.Name,
				SourceField:     c.Field,
				RelatedField:    c.RelatedField,
			}
			if sourceTableID > 0 {
				if fID, ok := fieldMap[fieldKey{sourceTableID, c.Field}]; ok {
					row.FkSourceField = &fID
				}
			}
			if relatedTableID > 0 {
				if fID, ok := fieldMap[fieldKey{relatedTableID, c.RelatedField}]; ok {
					row.FkRelatedField = &fID
				}
			}
			if c.SourceEDT != "" {
				row.SourceEdt = &c.SourceEDT
				if edtID, ok := edtMap[c.SourceEDT]; ok {
					row.FkSourceEdt = &edtID
				}
			}
			relFieldRows = append(relFieldRows, row)
		}
	}

	if len(relFieldRows) > 0 {
		if err := n.db.CreateInBatches(&relFieldRows, 1000).Error; err != nil {
			return 0, fmt.Errorf("inserting relation_fields: %w", err)
		}
	}

	// 13. Mark raw records as processed
	if err := n.db.Model(&model.RawAxTable{}).
		Where("id IN ?", rawIDs).
		Update("processed", true).Error; err != nil {
		return 0, fmt.Errorf("marking processed: %w", err)
	}

	return len(rawIDs), nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
