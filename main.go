package main

import (
	"fmt"
	"os"
	"sync"

	"r.tomng.dev/video2p8/image"
)

func main() {
	args := os.Args[1:]
	inputDir := ""
	outputDir := ""

	switch len(args) {
	case 1:
		inputDir = args[0]
	case 2:
		inputDir = args[0]
		outputDir = args[1]
	}

	if inputDir == "" {
		fmt.Print("Enter input directory: ")
		fmt.Scanln(&inputDir)
	}

	// Check if input directory exists
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		fmt.Println("Input directory does not exist")
		return
	}

	fmt.Println("Input directory: ", inputDir)

	if outputDir == "" {
		fmt.Print("Enter output directory: ")
		fmt.Scanln(&outputDir)
	}

	// Check if output directory exists
	// Re-create if it does
	frame_output_dir := fmt.Sprintf("%s%cframes", outputDir, os.PathSeparator)
	if _, err := os.Stat(frame_output_dir); err == nil {
		fmt.Print("Output directory is not empty. Do you want to REMOVE it? (Y/N) ")
		var remove_output rune
		fmt.Scanf("%c", &remove_output)

		if remove_output == 'Y' || remove_output == 'y' {
			os.RemoveAll(frame_output_dir)
		}
	}

	if err := os.MkdirAll(frame_output_dir, os.ModePerm); err != nil {
		fmt.Println("Error creating output directory: ", err)
		return
	}

	fmt.Println("Output directory: ", outputDir)

	// Get all JPG files in the input directory
	jpg_files, err := getJpgFiles(inputDir)
	if err != nil {
		fmt.Println("Error getting jpg files: ", err)
		return
	}

	// Create player cartridge
	writeP8Player(outputDir)

	// Process all JPG files to frames cartridges
	var wg sync.WaitGroup

	for i, jpg_file := range jpg_files {
		wg.Add(1)

		go func(jpg_file string) {
			defer wg.Done()

			fmt.Println("Processing ", jpg_file)
			pixelBytes, err := image.GetPixels(jpg_file)
			if err != nil {
				fmt.Println("Error getting pixels: ", err)
				return
			}

			if err := writeBytesToP8GFX(colourBytesToP8GfxBytes(pixelBytes), fmt.Sprintf("%s%c%d.p8", frame_output_dir, os.PathSeparator, i)); err != nil {
				panic(err)
			}
		}(jpg_file)
	}

	wg.Wait()
}

func getJpgFiles(input_dir string) ([]string, error) {
	files, err := os.ReadDir(input_dir)
	if err != nil {
		return nil, err
	}

	var jpg_files []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if file.Name()[len(file.Name())-4:] == ".jpg" {
			jpg_files = append(jpg_files, input_dir+string(os.PathSeparator)+file.Name())
		}
	}

	return jpg_files, nil
}

// Creates a new PICO-8 cartridge with the given bytes in the __gfx__ section
func writeBytesToP8GFX(bytes []byte, output_file string) error {
	f, err := os.Create(output_file)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write the PICO-8 code
	f.WriteString(`pico-8 cartridge // http://www.pico-8.com
version 42
__gfx__
`)

	lineCount := 0
	charCount := 0

	for _, b := range bytes {
		if charCount >= 128 {
			f.WriteString("\n")
			lineCount++
			charCount = 0
		}

		hex := reverse(fmt.Sprintf("%02x", b))
		f.WriteString(hex)
		charCount += len(hex)
	}

	for charCount < 128 {
		f.WriteString("0")
		charCount++
	}

	for (lineCount+1)%8 != 0 {
		f.WriteString("\n")
		for range 128 {
			f.WriteString("0")
		}
		lineCount++
	}

	return nil
}

// For each pair of bytes, combine them into a single byte in reverse order
// As each colour byte is 4 bits, we can combine them into a single byte
// E.g. 0x0a, 0x0b -> 0xba
func colourBytesToP8GfxBytes(bytes []byte) []byte {
	output := make([]byte, 0, len(bytes)/2)

	for i := 0; i < len(bytes); i += 2 {
		color1 := bytes[i]
		color2 := bytes[i+1]
		b := color1<<4 | color2

		output = append(output, b)
	}

	return output
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func writeP8Player(output_dir string) {
	f, err := os.Create(fmt.Sprintf("%s%cplayer.p8", output_dir, os.PathSeparator))
	if err != nil {
		fmt.Println("Error creating player.p8: ", err)
		return
	}
	defer f.Close()

	f.WriteString(`pico-8 cartridge // http://www.pico-8.com
version 42
__lua__
local frame = 0

function _init()
    poke(0x5f55, 0x00)
end

function _update60()
    frame += 1
    load_data(frame)
end

function load_data(i)
    reload(0, 0, 0x2000, "frames/" .. i .. ".p8")
end
`)
}
