package image

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
)

type FfmpegConfig struct {
	Fps              float32
	UsePalette       bool
	UsePaletteDither bool
	CropX            int
	CropY            int
	CropWidth        int
	CropHeight       int
	Brightness       float32
	Contrast         float32
	CutStart         int64
	CutEnd           int64
}

var Palette = [173]uint8{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0xF3, 0xFF, 0x61, 0x00, 0x00, 0x00,
	0x09, 0x70, 0x48, 0x59, 0x73, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x4F, 0x25, 0xC4, 0xD6, 0x00, 0x00, 0x00, 0x5F, 0x49, 0x44,
	0x41, 0x54, 0x78, 0x9C, 0x63, 0x60, 0x60, 0x60, 0xF8, 0xCF, 0xD0, 0x1E,
	0xF8, 0x9F, 0xE1, 0x89, 0xD9, 0x7F, 0x59, 0xED, 0xE0, 0xFF, 0x9A, 0x6B,
	0xFF, 0xFF, 0x8F, 0x0F, 0xF7, 0xFF, 0x5F, 0xA7, 0x1A, 0xFC, 0xBF, 0xB9,
	0x6C, 0xCE, 0xFF, 0xD5, 0x41, 0x66, 0xFF, 0x0F, 0x1D, 0x3E, 0xFE, 0xFF,
	0x3F, 0x83, 0xEF, 0xFF, 0xFF, 0xE5, 0x2B, 0xFE, 0xFF, 0x5F, 0xCC, 0xF0,
	0xFF, 0xFF, 0x99, 0x55, 0xFF, 0xFF, 0xBF, 0x51, 0xFF, 0xFF, 0xFF, 0xE3,
	0x8B, 0xFF, 0x0C, 0x20, 0x82, 0x12, 0x3C, 0x6A, 0xC0, 0xA8, 0x01, 0xA3,
	0x06, 0x0C, 0x06, 0x03, 0x80, 0x05, 0x01, 0x03, 0x00, 0x41, 0x7B, 0xBE,
	0xB2, 0xD7, 0xDD, 0xDB, 0x06, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
	0x44, 0xAE, 0x42, 0x60, 0x82,
}

func IsFFmpegSupported() bool {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return false
	}

	return true
}

// Converts provided video into a series of JPEG images
// which matches PICO-8's 128x128 resolution and black bar padding
// Returns true if the conversion was successful, false otherwise
func (ffmpegConfig *FfmpegConfig) ConvertVideoToJpeg(inputPath string, outputDir string) error {
	trimFilter := ""
	if ffmpegConfig.CutStart >= 0 && ffmpegConfig.CutEnd > 0 {
		trimFilter = fmt.Sprintf("trim=%d:%d,setpts=PTS-STARTPTS,", ffmpegConfig.CutStart, ffmpegConfig.CutEnd)
	}

	cropFilter := ""
	if ffmpegConfig.CropX >= 0 && ffmpegConfig.CropY >= 0 && ffmpegConfig.CropWidth > 0 && ffmpegConfig.CropHeight > 0 {
		cropFilter = fmt.Sprintf(",crop=%d:%d:%d:%d", ffmpegConfig.CropWidth, ffmpegConfig.CropHeight, ffmpegConfig.CropX, ffmpegConfig.CropY)
	}

	paletteFilter := ""
	if ffmpegConfig.UsePalette {
		paletteFilter = "[vid]; [vid][1:v]paletteuse"

		if !ffmpegConfig.UsePaletteDither {
			paletteFilter += "=dither=none"
		}
	}

	filters := fmt.Sprintf("[0:v]%sfps=%f%s,scale=128:128:force_original_aspect_ratio=decrease,eq=brightness=%f:contrast=%f,pad=128:128:-1:-1:color=black%s", trimFilter, ffmpegConfig.Fps, cropFilter, ffmpegConfig.Brightness, ffmpegConfig.Contrast, paletteFilter)

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-filter_complex", filters,
		outputDir+string(os.PathSeparator)+"%09d.jpg")

	if ffmpegConfig.UsePalette {
		cmd.Args = slices.Insert(cmd.Args, 3, "-i", "-")
		cmd.Stdin = bytes.NewBuffer(Palette[:])
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg output:\n", string(output))
		return err
	}

	return nil
}
