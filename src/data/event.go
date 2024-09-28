package data

type EventType int

const (
	EventNone EventType = iota
	EventEBPF
	EventPcap
)

type SnuffleEvent struct {
	Type EventType

	// It's not great that go does not support unions.  It's possible to do so
	// with a bunch of type assertions all over the place and 'any'
	//
	// Then you're just delegating all the heavy lifting to code that is more
	// elegantly done in data
	//
	// I choose to waste the 24 bytes on the sacrifical altar of code simplicity.
	//
	// If memory use, those 24 bytes, are shown to be a constraining factor then
	// I will reconsider this tradeoff.
	//
	// Either way?  Go does not support unions in the way that C does, and
	// does not allow you get away with memory murder the way that C does.
	//
	// Until measurement proves otherwise, 24 bytes per event is literally nothing.
	PcapEvent      *PcapEvent
	EBPFEvent      *EBPFEvent
	GNMIEvent      *GNMIEvent
	TelemetryEvent *TelemetryEvent
}

// Absent a union, or "unsafe", or a bunch of type assertions, this is
// the best we can do whilst making our code totally readable
func (e *SnuffleEvent) GetPcapEvent() *PcapEvent {
	if e.PcapEvent != nil {
		return e.PcapEvent
	}

	return nil
}

// Absent a union, or "unsafe", or a bunch of type assertions, this is
// the best we can do whilst making our code totally readable
func (e *SnuffleEvent) GetEBPFEvent() *EBPFEvent {
	if e.EBPFEvent != nil {
		return e.EBPFEvent
	}

	return nil
}

// Absent a union, or "unsafe", or a bunch of type assertions, this is
// the best we can do whilst making our code totally readable
func (e *SnuffleEvent) GetGNMIEvent() *GNMIEvent {
	if e.GNMIEvent != nil {
		return e.GNMIEvent
	}

	return nil
}

// Absent a union, or "unsafe", or a bunch of type assertions, this is
// the best we can do whilst making our code totally readable
func (e *SnuffleEvent) GetTelemetryEvent() *TelemetryEvent {
	if e.TelemetryEvent != nil {
		return e.TelemetryEvent
	}

	return nil
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

// TODO: replace with actual GNMI struct
type GNMIEvent struct{}

// TODO: replace with actual telemetry struct
type TelemetryEvent struct{}
