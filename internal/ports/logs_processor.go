package ports

type Processor interface {
	Process() error
	Stop()
}
