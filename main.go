package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"r.tomng.dev/video2p8/image"
)

type WorkerInput struct {
	JpgFile string
	Index   int
}

var (
	ffmpegConfig = image.FfmpegConfig{}
	inputVideo   string
	outputDir    string
	autorunP8    bool
	workers      int
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
	flags.IntVarP(&workers, "workers", "w", 1000, "Number of workers to process frames.")

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
	flags.Int64Var(&ffmpegConfig.CutStart, "cut-start", 0, "Cut start timestamp in seconds. E.g. 69")
	flags.Int64Var(&ffmpegConfig.CutEnd, "cut-end", 0, "Cut end timestamp in seconds. E.g. 420")
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

		if remove_output == 'Y' || remove_output == 'y' || remove_output == '\r' {
			if err := os.RemoveAll(jpegFramesOutputDir); err != nil {
				fmt.Println("Error removing output directory:", err)
				return
			}
			if err := os.RemoveAll(framesOutputDir); err != nil {
				fmt.Println("Error removing output directory:", err)
				return
			}
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
	fmt.Println("Converting video to JPG frames...")
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
	jobs := make(chan *WorkerInput, len(jpg_files))

	// Start workers
	for range workers {
		wg.Add(1)
		go worker(&wg, jobs, framesOutputDir)
	}

	for i, jpg_file := range jpg_files {
		jobs <- &WorkerInput{JpgFile: jpg_file, Index: i}
	}
	close(jobs)

	wg.Wait()

	if autorunP8 {
		runP8Player(outputDir)
	}
}

func worker(wg *sync.WaitGroup, jobs <-chan *WorkerInput, framesOutputDir string) {
	defer wg.Done()

	for job := range jobs {
		fmt.Println("Processing", job.JpgFile)
		colourBytes, err := image.GetP8Colours(job.JpgFile)
		if err != nil {
			fmt.Println("Error getting pixels:", err)
			return
		}

		cartCount := -32768.0 + float64(job.Index)*0.0001

		if err := writeBytesToP8GFX(colourBytesToP8GfxBytes(colourBytes), fmt.Sprintf("%s%c%s.p8", framesOutputDir, os.PathSeparator, strconv.FormatFloat(cartCount, 'f', -1, 64))); err != nil {
			panic(err)
		}
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

// For each pair of bytes, combine them into a single byte
// As each colour byte is 4 bits, we can combine them into a single byte
// E.g. [0] -> 0x0a, [1] -> 0x0b = 0xba
func colourBytesToP8GfxBytes(bytes []byte) []byte {
	output := make([]byte, 0, len(bytes)/2)

	for i := 0; i < len(bytes); i += 2 {
		color1 := bytes[i]
		color2 := bytes[i+1]
		b := color2<<4 | color1

		output = append(output, b)
	}

	return output
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
	code := `pico-8 cartridge // http://www.pico-8.com
version 42
__gfx__`

	lineCount := 0

	for i := 0; i < len(bytes); i += 64 {
		code += "\n" + hex.EncodeToString(bytes[i:i+64])
		lineCount++
	}

	bytesWritten := lineCount * 64
	bytesLeft := len(bytes) - bytesWritten
	if bytesLeft != 0 {
		lineCount++
	}

	f.WriteString(code +
		"\n" + hex.EncodeToString(bytes[bytesWritten:]) +
		strings.Repeat("00", 64-bytesLeft) +
		strings.Repeat("\n00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", lineCount%8),
	)

	return nil
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
local f = -32768.0
local s = 1

function _draw()
 if s == 0 then
  print("paused",0,122,7)
 elseif s != 1 then
  print("speed: x"..s,0,122,7)
 end
end

function _update60()
 if btnp(⬅️) then
  s -= (s == 1 and 2 or 1)
 elseif btnp(➡️) then
  s += (s == -1 and 2 or 1)
 elseif btnp(❎) then
  s = (s == 0 and 1 or 0)
 end
 if s != 0 then
  f += 0.0001 * s
  reload(0x6000, 0, 0x2000, "frames/" .. f .. ".p8")
 end
end
`)
}

func runP8Player(output_dir string) {
	cmd := exec.Command("pico8", "-run", fmt.Sprintf("%s%cplayer.p8", output_dir, os.PathSeparator))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
}
