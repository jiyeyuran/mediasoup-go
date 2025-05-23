// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package Transport

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type PortRangeT struct {
	Min uint16 `json:"min"`
	Max uint16 `json:"max"`
}

func (t *PortRangeT) Pack(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	if t == nil {
		return 0
	}
	PortRangeStart(builder)
	PortRangeAddMin(builder, t.Min)
	PortRangeAddMax(builder, t.Max)
	return PortRangeEnd(builder)
}

func (rcv *PortRange) UnPackTo(t *PortRangeT) {
	t.Min = rcv.Min()
	t.Max = rcv.Max()
}

func (rcv *PortRange) UnPack() *PortRangeT {
	if rcv == nil {
		return nil
	}
	t := &PortRangeT{}
	rcv.UnPackTo(t)
	return t
}

type PortRange struct {
	_tab flatbuffers.Table
}

func GetRootAsPortRange(buf []byte, offset flatbuffers.UOffsetT) *PortRange {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &PortRange{}
	x.Init(buf, n+offset)
	return x
}

func FinishPortRangeBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsPortRange(buf []byte, offset flatbuffers.UOffsetT) *PortRange {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &PortRange{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedPortRangeBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *PortRange) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *PortRange) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *PortRange) Min() uint16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetUint16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *PortRange) MutateMin(n uint16) bool {
	return rcv._tab.MutateUint16Slot(4, n)
}

func (rcv *PortRange) Max() uint16 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetUint16(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *PortRange) MutateMax(n uint16) bool {
	return rcv._tab.MutateUint16Slot(6, n)
}

func PortRangeStart(builder *flatbuffers.Builder) {
	builder.StartObject(2)
}
func PortRangeAddMin(builder *flatbuffers.Builder, min uint16) {
	builder.PrependUint16Slot(0, min, 0)
}
func PortRangeAddMax(builder *flatbuffers.Builder, max uint16) {
	builder.PrependUint16Slot(1, max, 0)
}
func PortRangeEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
