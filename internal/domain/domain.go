package domain

const Selector = "mta"

type Domain struct {
	ID string
	Name string
	HasSPF bool
	SPFRecord string
	HasDKIM bool
	HasDMARC bool
	DMARCRecord string
}
