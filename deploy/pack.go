package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"os"

	"atari/engine"
)

const WeightFile = "../weights.bin"

func main() {
	data, err := os.ReadFile(WeightFile)
	if err != nil {
		fmt.Println("❌ Ошибка: не найден файл weights.bin в корне.")
		return
	}

	if len(data) != engine.TotalParam*8 {
		fmt.Println("❌ Ошибка: размер файла не совпадает с float64.")
		return
	}

	buf32 := make([]byte, engine.TotalParam*4)
	for i := 0; i < engine.TotalParam; i++ {
		bits64 := binary.LittleEndian.Uint64(data[i*8 : (i+1)*8])
		val32 := float32(math.Float64frombits(bits64))
		bits32 := math.Float32bits(val32)
		binary.LittleEndian.PutUint32(buf32[i*4:(i+1)*4], bits32)
	}

	encoded := base64.StdEncoding.EncodeToString(buf32)
	fmt.Println("\n=== СТРОКА ДЛЯ CODINGAME ===")
	fmt.Println(encoded)
	fmt.Println("=============================")
}
