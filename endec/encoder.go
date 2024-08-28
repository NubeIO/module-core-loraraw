package endec

func reshapeVector(serialData *SerialData, bitCount int, dataVector *[]byte) {
	// Check if extra byte is required
	if (serialData.WriteBitPos + bitCount) > len(*dataVector)*8 {
		*dataVector = append(*dataVector, 0)
	}

	// Reshape vector to match buffer
	var TB uint16 = 0
	for i := range *dataVector {
		TB = ((TB | uint16((*dataVector)[i])) << 8) >> serialData.WriteBitPos
		(*dataVector)[i] = byte(TB >> 8)
		TB = TB << serialData.WriteBitPos
	}
}
