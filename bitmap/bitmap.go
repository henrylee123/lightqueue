package bitmap

import "sync"

type Bitmap struct {
	data      []byte
	size      uint64
	resetLock sync.Mutex
}

func (b *Bitmap) Init(size uint64) {
	size = (size + 7) / 8 * 8
	b.size = size
	b.data = make([]uint8, size/8, size/8)
}

func (b *Bitmap) Reset(sizek uint64) {
	b.resetLock.Lock()
	defer b.resetLock.Unlock()

	for i := 0; i < len(b.data); i++ {
		b.data[i] = 0
	}
}

func (b *Bitmap) Set(pos uint64) bool {
	if pos >= b.size {
		return false
	}
	b.data[pos>>3] |= 1 << (pos & 0x07)
	return true
}

func (b *Bitmap) SetM(posStart uint64, posEnd uint64) bool {
	if posStart >= b.size || posEnd >= b.size {
		return false
	}
	for i := posStart; i <= posEnd; i++ {
		b.data[i>>3] |= 1 << (i & 0x07)
	}
	return true
}

func (b *Bitmap) Unset(pos uint64) bool {
	if pos >= b.size {
		return false
	}
	b.data[pos>>3] &= ^(1 << (pos & 0x07))
	return true
}

func (b *Bitmap) IsSet(pos uint64) bool {
	if pos >= b.size {
		return false
	}
	if b.data[pos>>3]&(1<<(pos&0x07)) > 0 {
		return true
	}
	return false
}

func (b *Bitmap) String() string {
	var s string
	for i := uint64(1); i < b.size+1; i++ {
		if b.IsSet(i - 1) {
			s += "1"
		} else {
			s += "0"
		}
		if i%10 == 0 {
			s += " "
		}
	}
	return s
}
