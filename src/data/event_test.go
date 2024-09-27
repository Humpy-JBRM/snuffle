package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPcapEvent(t *testing.T) {
	pcapEvent := &PcapEvent{}
	snuffleEvent := NewPcapEvent(pcapEvent)

	assert.Equal(t, snuffleEvent.Type, EventPcap)
	assert.Equal(t, snuffleEvent.PcapEvent, pcapEvent)
}

func TestNewEBPFEvent(t *testing.T) {
	ebpfEvent := &EBPFEvent{}
	snuffleEvent := NewEBPFEvent(ebpfEvent)

	assert.Equal(t, snuffleEvent.Type, EventEBPF)
	assert.Equal(t, snuffleEvent.EBPFEvent, ebpfEvent)
}
