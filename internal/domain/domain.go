package domain

const Selector = "my_mta"

type Domain struct {
	ID string
	Name string
	HasSPF bool
	SPFRecord string
	HasDKIM bool
	HasDMARC bool
	DMARCRecord string
}
