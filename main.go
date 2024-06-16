package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"r.tomng.dev/video2p8/image"
)

var (
	ffmpegConfig = image.FfmpegConfig{}
	inputVideo   string
	outputDir    string
	autorunP8    bool
)

var rootCmd = &cobra.Command{
	Use:     "video2p8 [flags] -i <input_video> -o <out_dir>",
	Example: "video2p8 -i video.mp4 -o output_dir --contrast 1.5",
	Short:   "Converts a video into PICO-8 cartridges.",
	Long:    "A tool to convert a video using ffmpeg into a series of PICO-8 cartridges, represent each frame of the video. A player cartridge is also created to play the frames in PICO-8.",
	Run:     execute,
}

func init() {
	flags := rootCmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&inputVideo, "input", "i", "", "Input video file")
	rootCmd.MarkFlagRequired("input")

	flags.StringVarP(&outputDir, "output", "o", "", "Output directory")
	rootCmd.MarkFlagRequired("output")

	flags.BoolVar(&autorunP8, "autorun", false, "Autorun the player cartridge after conversion. Only works if \"pico8\" is in PATH")

	// FFmpeg configs
	flags.Float32Var(&ffmpegConfig.Fps, "fps", 19.89, "Frames per second")
	flags.BoolVar(&ffmpegConfig.UsePalette, "use-palette", false, "Use palette")
	flags.BoolVar(&ffmpegConfig.UsePaletteDither, "use-palette-dither", false, "Use palette dither")
	flags.IntVar(&ffmpegConfig.CropX, "cx", 0, "Crop X")
	flags.IntVar(&ffmpegConfig.CropY, "cy", 0, "Crop Y")
	flags.IntVar(&ffmpegConfig.CropWidth, "cw", 0, "Crop width")
	flags.IntVar(&ffmpegConfig.CropHeight, "ch", 0, "Crop height")
	flags.Float32Var(&ffmpegConfig.Brightness, "brightness", 0, "Brightness")
	flags.Float32Var(&ffmpegConfig.Contrast, "contrast", 1, "Contrast")
}

func main() {
	if !image.IsFFmpegSupported() {
		fmt.Println("ffmpeg is not installed or not in PATH")
		return
	}

	rootCmd.Execute()
}

func execute(cmd *cobra.Command, args []string) {
	framesOutputDir, err := filepath.Abs(fmt.Sprintf("%s%cframes", outputDir, os.PathSeparator))
	jpegFramesOutputDir, err2 := filepath.Abs(fmt.Sprintf("%s%craw", outputDir, os.PathSeparator))
	if err != nil {
		fmt.Println("Error getting absolute path:", err)
		return
	}
	if err2 != nil {
		fmt.Println("Error getting absolute path:", err2)
		return
	}

	// Check if output directories exists
	// Re-create they do
	_, err = os.Stat(jpegFramesOutputDir)
	_, err2 = os.Stat(framesOutputDir)

	if err == nil || err2 == nil {
		fmt.Print("Output directory is not empty. Do you want to REMOVE it? (Y/n) ")
		var remove_output rune
		fmt.Scanf("%c", &remove_output)

		if remove_output == 'Y' || remove_output == 'y' || remove_output == '\n' {
			os.RemoveAll(jpegFramesOutputDir)
			os.RemoveAll(framesOutputDir)
		}
	}

	if err := os.MkdirAll(jpegFramesOutputDir, os.ModePerm); err != nil {
		fmt.Println("Error creating output directory:", err)
		return
	}

	if err := os.MkdirAll(framesOutputDir, os.ModePerm); err != nil {
		fmt.Println("Error creating output directory:", err)
		return
	}

	// Converts video to JPG frames
	if err := ffmpegConfig.ConvertVideoToJpeg(inputVideo, jpegFramesOutputDir); err != nil {
		fmt.Println("Error converting video to JPG:", err)
		return
	}

	// Get all JPG files in the input directory
	jpg_files, err := getJpgFiles(jpegFramesOutputDir)
	if err != nil {
		fmt.Println("Error getting jpg files:", err)
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

			fmt.Println("Processing", jpg_file)
			colourBytes, err := image.GetP8Colours(jpg_file)
			if err != nil {
				fmt.Println("Error getting pixels:", err)
				return
			}

			if err := writeBytesToP8GFX(colourBytesToP8GfxBytes(colourBytes), fmt.Sprintf("%s%c%d.p8", framesOutputDir, os.PathSeparator, i)); err != nil {
				panic(err)
			}
		}(jpg_file)
	}

	wg.Wait()

	if autorunP8 {
		runP8Player(outputDir)
	}
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
// Bytes are reversed then written from left to right
// E.g. Screen with colour codes: 12 34
// Will be written as: 21 43
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

	for ; (lineCount+1)%8 != 0; lineCount++ {
		f.WriteString("\n")
		for range 128 {
			f.WriteString("0")
		}
	}

	return nil
}

// For each pair of bytes, combine them into a single byte
// As each colour byte is 4 bits, we can combine them into a single byte
// E.g. 0x0a, 0x0b -> 0xab
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
		fmt.Println("Error creating player.p8:", err)
		return
	}
	defer f.Close()

	f.WriteString(`pico-8 cartridge // http://www.pico-8.com
version 42
__lua__
local f = 0

function _update60()
 f += 1
 reload(0x6000, 0, 0x2000, "frames/" .. f .. ".p8")
end
`)
}

func runP8Player(output_dir string) {
	cmd := exec.Command("pico8", "-run", fmt.Sprintf("%s%cplayer.p8", output_dir, os.PathSeparator))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
}
