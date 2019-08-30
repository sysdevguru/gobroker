package encryption

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

var (
	mByteOnBits = []int{1, 3, 7, 15, 31, 63, 127, 255}
	mByte2Power = []int{1, 2, 4, 8, 16, 32, 64, 128}
	mInCo       = []int{0xB, 0xD, 0x9, 0xE}
	mIntOnBits  = []int{1, 3, 7, 15, 31, 63, 127, 255, 511, 1023, 2047, 4095, 8191, 16383, 32767,
		65535, 131071, 262143, 524287, 1048575, 2097151, 4194303, 8388607,
		16777215, 33554431, 67108863, 134217727, 268435455, 536870911,
		1073741823, 2147483647}
	mInt2Power = []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384,
		32768, 65536, 131072, 262144, 524288, 1048576, 2097152, 4194304,
		8388608, 16777216, 33554432, 67108864, 134217728, 268435456,
		536870912, 1073741824}
	mLTab                                         []int16
	mPTab, mFBSub, mRBSub, mRTable, mFTable, mRCo []int
	mNk, mNb, mNr                                 int
	mFI, mFKey                                    []int
)

func rijndaelEncrypt(pText string, pEncryptionKey string) (string, error) {
	if &pText == nil || pText == "" {
		return "", fmt.Errorf("Nil or empty text passed into Encryption.rijndaelEncrypt.")
	} else if &pEncryptionKey == nil || pEncryptionKey == "" {
		return "", fmt.Errorf("Nil or empty encryption key passed into Encryption.rijndaelEncrypt.")
	}

	pTextBytes := []byte(pText)
	pEncryptionKeyBytes := []byte(pEncryptionKey)
	lEncryptedBytes := encryptData(&pTextBytes, &pEncryptionKeyBytes)

	hexEncryptionString := getHexString(lEncryptedBytes)
	return hexEncryptionString, nil
}

func getHexString(pBytes []int) string {
	var lHexString bytes.Buffer
	for i := 0; i < len(pBytes); i++ {
		if pBytes[i] < 16 {
			lHexString.WriteString("0")
		}
		lHexString.WriteString(fmt.Sprintf("%X", pBytes[i]))
	}
	return lHexString.String()
}

// Since packFrom originally used overloading, we just use two seperate functions since golang doesn't support overloading
func packFromII(pB *[]int, k int) int {
	lResult := 0
	pBResult := *pB
	for lCount := 0; lCount < 4; lCount++ {
		lResult = lResult | leftShiftInt(pBResult[lCount+k], (lCount*8))
	}
	return lResult
}

func packFromBI(pB *[]byte, k int) int {
	lResult := 0
	pBResult := *pB

	for lCount := 0; lCount < 4; lCount++ {
		lResult = lResult | leftShiftInt(int(pBResult[lCount+k]), (lCount*8))
	}
	return lResult
}

func subByte(a int) int {
	if mFBSub == nil {
		generateTables()
	}
	b := make([]int, 4)
	unpack(a, &b)
	b[0] = mFBSub[b[0]]
	b[1] = mFBSub[b[1]]
	b[2] = mFBSub[b[2]]
	b[3] = mFBSub[b[3]]
	return pack(&b)
}

func gkey(pNb int, pNk int, pKey *[]byte) {
	mNb = pNb
	mNk = pNk
	mNr = int(6 + math.Max(float64(mNb), float64(mNk)))

	c1 := 1
	c2 := 3
	c3 := 4

	if mNb < 8 {
		c2 = 2
		c3 = 3
	}
	mFI = make([]int, 24)
	mRI := make([]int, 24)
	var j int
	for j = 0; j < pNb; j++ {
		m := j * 3

		mFI[m] = (j + c1) % pNb
		mFI[m+1] = (j + c2) % pNb
		mFI[m+2] = (j + c3) % pNb
		mRI[m] = (pNb + j - c1) % pNb
		mRI[m+1] = (pNb + j - c2) % pNb
		mRI[m+2] = (pNb + j - c3) % pNb
	}

	N := mNb * (mNr + 1)

	mFKey = make([]int, 120)
	lCipherKey := make([]int, 8)

	for i := 0; i < mNk; i++ {
		j = i * 4
		lCipherKey[i] = packFromBI(pKey, j)
		mFKey[i] = packFromBI(pKey, j)
	}

	if mRCo == nil {
		generateTables()
	}

	j = mNk
	for k := 0; j < N; k++ {
		mFKey[j] = mFKey[j-mNk] ^ subByte(rotateLeftInt(mFKey[j-1], 24)) ^ mRCo[k]
		if mNk <= 6 {
			for i := 1; i < mNk && (i+j) < N; i++ {
				mFKey[i+j] = mFKey[i+j-mNk] ^ mFKey[i+j-1]
			}
		} else {
			for i := 1; i < 4 && (i+j) < N; i++ {
				mFKey[i+j] = mFKey[i+j-mNk] ^ mFKey[i+j-1]
			}
			if j+4 < N {
				mFKey[j+4] = mFKey[j+4-mNk] ^ subByte(mFKey[j+3])
			}
			for i := 5; i < mNk && (i+j) < N; i++ {
				mFKey[i+j] = mFKey[i+j-mNk] ^ mFKey[i+j-1]
			}
		}
		j = j + mNk
	}

	mRKey := make([]int, 120)

	for j := 0; j < mNk-1; j++ {
		mRKey[j+N-pNb] = mFKey[j]
	}

	for i := mNb; i < N-mNk; i += mNk {
		k := N - mNk - i
		for j := 0; j < mNk-1; j++ {
			mRKey[k+j] = invMixCol(mFKey[i+j])
		}
	}

	for j := N - mNk; j < N; j++ {
		mRKey[j-N+mNb] = mFKey[j]
	}
}

func product(x int, y int) int {
	xb := make([]int, 4)
	yb := make([]int, 4)

	unpack(x, &xb)
	unpack(y, &yb)

	return (bmul(xb[0], yb[0]) ^ bmul(xb[1], yb[1]) ^ bmul(xb[2], yb[2]) ^ bmul(xb[3], yb[3]))
}

func invMixCol(x int) int {
	b := make([]int, 4)
	m := pack(&mInCo)
	b[3] = product(m, x)
	m = rotateLeftInt(m, 24)
	b[2] = product(m, x)
	m = rotateLeftInt(m, 24)
	b[1] = product(m, x)
	m = rotateLeftInt(m, 24)
	b[0] = product(m, x)
	return pack(&b)
}

func encryptData(pData *[]byte, pPassword *[]byte) []int {
	lKey := make([]byte, 32)
	tempPassword := *pPassword
	for lCount := 0; lCount < int(math.Min(float64(len(*pPassword)), 32.0)); lCount++ {
		lKey[lCount] = tempPassword[lCount]
	}

	gkey(8, 8, &lKey)

	lLength := len(*pData)
	lEncodedLength := lLength + 4

	if lEncodedLength%32 != 0 {
		lEncodedLength = lEncodedLength + 32 - (lEncodedLength % 32)
	}

	lInBytes := make([]int, lEncodedLength)
	unpack(lLength, &lInBytes)
	copyBytesASP(&lInBytes, 4, pData, 0, lLength)

	lTempBytes := make([]int, 32)
	lOutBytes := make([]int, lEncodedLength)
	for lCount := 0; lCount < lEncodedLength-1; lCount += 32 {
		copyBytesASP2(&lTempBytes, 0, &lInBytes, lCount, 32)
		encrypt(&lTempBytes)
		copyBytesASP2(&lOutBytes, lCount, &lTempBytes, 0, 32)
	}

	return lOutBytes

}

func encrypt(pBuffer *[]int) {
	x := make([]int, 8)
	y := make([]int, 8)

	for i := 0; i < mNb; i++ {
		j := i * 4
		x[i] = packFromII(pBuffer, j) ^ mFKey[i]
	}

	k := mNb
	t := make([]int, 0)

	if mFTable == nil {
		generateTables()
	}

	for i := 1; i < mNr; i++ {
		for j := 0; j < mNb; j++ {
			m := j * 3
			y[j] = mFKey[k] ^ mFTable[x[j]&mIntOnBits[7]] ^
				rotateLeftInt(mFTable[rightShiftInt(x[mFI[m]], 8)&mIntOnBits[7]], 8) ^
				rotateLeftInt(mFTable[rightShiftInt(x[mFI[m+1]], 16)&mIntOnBits[7]], 16) ^
				rotateLeftInt(mFTable[rightShiftInt(x[mFI[m+2]], 24)&mIntOnBits[7]], 24)
			k += 1
		}
		t = x
		x = y
		y = t
	}

	for j := 0; j < mNb; j++ {
		m := j * 3
		y[j] = mFKey[k] ^ mFBSub[x[j]&mIntOnBits[7]] ^
			rotateLeftInt(mFBSub[rightShiftInt(x[mFI[m]], 8)&mIntOnBits[7]], 8) ^
			rotateLeftInt(mFBSub[rightShiftInt(x[mFI[m+1]], 16)&mIntOnBits[7]], 16) ^
			rotateLeftInt(mFBSub[rightShiftInt(x[mFI[m+2]], 24)&mIntOnBits[7]], 24)
		k++
	}

	for i := 0; i < mNb; i++ {
		j := i * 4
		unpackFrom(y[i], *pBuffer, j)
		x[i] = 0
		y[i] = 0
	}
}

// type pDest []int
// type pSource []byte

func copyBytesASP(pDestination *[]int, pDestinationStart int, pSource *[]byte, pSourceStart int, pLength int) {
	pDestinationArray := *pDestination
	pSourceArray := *pSource
	for lCount := 0; lCount < pLength; lCount++ {
		pDestinationArray[pDestinationStart+lCount] = int(pSourceArray[pSourceStart+lCount])
	}
}

func copyBytesASP2(pDestination *[]int, pDestinationStart int, pSource *[]int, pSourceStart int, pLength int) {
	pDestinationArray := *pDestination
	pSourceArray := *pSource
	for lCount := 0; lCount < pLength; lCount++ {
		pDestinationArray[pDestinationStart+lCount] = pSourceArray[pSourceStart+lCount]
	}
}

func leftShiftInt(pValue int, pShiftBits int) int {
	// Since 0x80000000 in Go has a different value than it has in Java if just used directly
	// Java => -2147483648
	// Go => 2147483648
	// this is an overflow issue
	tempBitU := uint32(0x80000000)
	tempBit := int32(tempBitU)

	if pShiftBits < 0 || pShiftBits > 31 {
		fmt.Printf("Shift bits must be between 0 and 31.  Passed: %v", pShiftBits)
		os.Exit(1)
	}
	if pShiftBits == 0 {
		return pValue
	}
	if pShiftBits == 31 {
		if (pValue & 1) != 0 {
			return int(tempBit)
		}
		return 0
	}
	if (pValue & mInt2Power[31-pShiftBits]) != 0 {
		return ((pValue & mIntOnBits[31-(pShiftBits+1)]) * mInt2Power[pShiftBits]) | int(tempBit)
	}
	return (pValue & mIntOnBits[31-pShiftBits]) * mInt2Power[pShiftBits]
}

func rightShiftInt(pValue int, pShiftBits int) int {
	tempBitU := uint32(0x80000000)
	tempBit := int32(tempBitU)
	if pShiftBits < 0 || pShiftBits > 31 {
		fmt.Printf("Shift bits must be between 0 and 31.  Passed: %v", pShiftBits)
		os.Exit(1)
	}
	if pShiftBits == 0 {
		return pValue
	}
	if pShiftBits == 31 {
		if (pValue & int(tempBit)) != 0 {
			return 1
		}
		return 0
	}
	// other bits handled correctly
	lResult := (pValue & 0x7FFFFFFE) / mInt2Power[pShiftBits]
	if (pValue & int(tempBit)) != 0 {
		lResult = lResult | (0x40000000 / mInt2Power[pShiftBits-1])
	}
	return lResult
}

func leftShiftByte(pValue int, pShiftBits int) int {
	if pShiftBits < 0 || pShiftBits > 7 {
		fmt.Printf("Shift bits must be between 0 and 7.  Passed: %v", pShiftBits)
		os.Exit(1)
	}
	if pShiftBits == 0 {
		return pValue
	}
	if pShiftBits == 7 {
		if (pValue & 1) != 0 {
			return 0x80
		}
		return 0
	}
	return (pValue & mByteOnBits[7-pShiftBits]) * mByte2Power[pShiftBits]
}

func rightShiftByte(pValue int, pShiftBits int) int {
	if pShiftBits < 0 || pShiftBits > 7 {
		fmt.Printf("Shift bits must be between 0 and 7.  Passed: %v", pShiftBits)
		os.Exit(1)
	}
	if pShiftBits == 0 {
		return pValue
	}
	if pShiftBits == 7 {
		if (pValue & 0x80) != 0 {
			return 1
		}
		return 0
	}
	return pValue / mByte2Power[pShiftBits]
}

func rotateLeftInt(pValue int, pShiftBits int) int {
	return leftShiftInt(pValue, pShiftBits) | rightShiftInt(pValue, 32-pShiftBits)
}

func rotateLeftByte(pValue int, pShiftBits int) int {
	return leftShiftByte(pValue, pShiftBits) | rightShiftByte(pValue, 8-pShiftBits)
}

func pack(pB *[]int) int {
	var lResult int
	lResult = 0
	tempPB := *pB
	var lCount byte
	var lTemp int
	for lCount = 0; lCount < 4; lCount++ {
		lTemp = tempPB[lCount]
		lResult = lResult | leftShiftInt(lTemp, int(lCount*8))
	}
	return lResult
}

func unpack(a int, pB *[]int) {
	tempPB := *pB
	tempPB[0] = a & mIntOnBits[7]
	tempPB[1] = rightShiftInt(a, 8) & mIntOnBits[7]
	tempPB[2] = rightShiftInt(a, 16) & mIntOnBits[7]
	tempPB[3] = rightShiftInt(a, 24) & mIntOnBits[7]
}

func unpackFrom(a int, pB []int, k int) {
	pB[0+k] = a & mIntOnBits[7]
	pB[1+k] = rightShiftInt(a, 8) & mIntOnBits[7]
	pB[2+k] = rightShiftInt(a, 16) & mIntOnBits[7]
	pB[3+k] = rightShiftInt(a, 24) & mIntOnBits[7]
}

func xtime(pIn int) int {
	b := 0

	if (pIn & 0x80) != 0 {
		b = 0x1B
	}

	return leftShiftByte(pIn, 1) ^ b
}

func bmul(pX int, pY int) int {
	if mPTab == nil {
		generateTables()
	}

	if pX != 0 && pY != 0 {
		return mPTab[(mLTab[pX]+mLTab[pY])%255]
	}

	return 0
}

func byteSub(pIn int) int {
	if mPTab == nil {
		generateTables()
	}

	z := pIn
	y := mPTab[255-mLTab[z]]
	z = y
	z = rotateLeftByte(z, 1)
	y = y ^ z
	z = rotateLeftByte(z, 1)
	y = y ^ z
	z = rotateLeftByte(z, 1)
	y = y ^ z
	z = rotateLeftByte(z, 1)
	y = y ^ z
	y = y ^ 0x63
	return y
}

func generateTables() {
	mLTab = make([]int16, 256)
	mPTab = make([]int, 256)
	mFBSub = make([]int, 256)
	mRBSub = make([]int, 256)
	mFTable = make([]int, 256)
	mRTable = make([]int, 256)
	mRCo = make([]int, 256)
	var y int

	mLTab[0] = 0
	mPTab[0] = 1
	mLTab[1] = 0
	mPTab[1] = 3
	mLTab[3] = 1

	// This isn't done correctly => casting
	var i int16
	for i = 2; i < 256; i++ {
		temp := mPTab[i-1] ^ xtime(mPTab[i-1])
		mPTab[i] = temp
		mLTab[mPTab[i]] = int16(i)
	}

	mFBSub[0] = 0x63
	mRBSub[0x63] = 0

	for i = 1; i < 256; i++ {
		ib := int(i)
		y = byteSub(ib)
		mFBSub[i] = y
		mRBSub[y] = int(i)
	}

	y = 1
	for i := 0; i < 30; i++ {
		mRCo[i] = y
		y = xtime(y)
	}

	b := make([]int, 4)
	for i := 0; i < 256; i++ {
		y = mFBSub[i]
		b[3] = y ^ xtime(y)
		b[2] = y
		b[1] = y
		b[0] = xtime(y)
		mFTable[i] = pack(&b)

		y = mRBSub[i]
		b[3] = bmul(mInCo[0], y)
		b[2] = bmul(mInCo[1], y)
		b[1] = bmul(mInCo[2], y)
		b[0] = bmul(mInCo[3], y)
		mRTable[i] = pack(&b)
	}
}

func GenRandomKey(length int) string {
	b := make([]byte, 2*length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%X", b)[0:length]
}

func EncryptedTimestamp() (*string, error) {
	timeStamp := time.Now().UTC().Format("2006/01/02 15:04")
	key := os.Getenv("APEX_ENCRYPTION_KEY")

	output, err := rijndaelEncrypt(timeStamp,
		key)
	if err != nil {
		return nil, err
	}
	encrypted := string(output)
	return &encrypted, nil
}
