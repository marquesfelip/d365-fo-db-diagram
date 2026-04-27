package pipeline

type RawRecord struct {
	Name  string
	Model string
	Layer string
	Data  []byte
}
