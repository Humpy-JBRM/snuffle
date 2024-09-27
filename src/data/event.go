package data

type EventType int

const (
	EventNone EventType = iota
	EventEBPF
	EventPcap
)

type SnuffleEvent struct {
	Type      EventType
	PcapEvent *PcapEvent
	EBPFEvent *EBPFEvent
}

func NewPcapEvent(pcapEvent *PcapEvent) *SnuffleEvent {
	return &SnuffleEvent{
		Type:      EventPcap,
		PcapEvent: pcapEvent,
	}
}

func NewEBPFEvent(ebpfEvent *EBPFEvent) *SnuffleEvent {
	return &SnuffleEvent{
		Type:      EventEBPF,
		EBPFEvent: ebpfEvent,
	}
}

// TODO: replace with actual pcap data struct
type PcapEvent struct{}

// TODO: replace with actual EBPF struct
type EBPFEvent struct{}
