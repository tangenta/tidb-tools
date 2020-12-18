package cluster_index

type ControlOption struct {
	MaxTableNum int
}

func DefaultControlOption() *ControlOption {
	return &ControlOption{
		MaxTableNum: 5,
	}
}
