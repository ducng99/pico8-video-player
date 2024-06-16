# video2p8

A tool to convert a video using ffmpeg into a series of PICO-8 cartridges, represent each frame of the video. A player cartridge is also created to play the frames in PICO-8.

The video will be extracted into 128x128 JPEG frames then converted into .p8 carts. The player cartridge then loads GFX from these "frame" carts continuosly to display the video.

## Requirements

[FFmpeg](https://www.ffmpeg.org/download.html) must be installed and accessible in PATH as `ffmpeg`

## Usage

The app takes in a video and a path to an empty directory for output. If the output directory is not empty, it will prompt if you want to remove it.

```sh
video2p8 -i ./video.mp4 -o ./outputp8
```

Some ffmpeg filters are customisable that may or may not provide a better result, such as contrast and FPS.
Crop can be helpful to remove black bars in the original video.

Run with `-h` or `--help` to get a full list of flags available.

```sh
$ video2p8 -h

A tool to convert a video using ffmpeg into a series of PICO-8 cartridges, represeting each frame of the video. A player cartridge is also created to play the frames in PICO-8.

Usage:
  video2p8 [flags] -i <input_video> -o <out_dir>

Examples:
video2p8 -i video.mp4 -o output_dir --contrast 1.5

Flags:
  -i, --input string         Input video file
  -o, --output string        Output directory
      --autorun              Autorun the player cartridge after conversion. Only works if "pico8" is in PATH
      --fps float32          Frames per second (default 19.89)
      --use-palette          Use palette
      --use-palette-dither   Use palette dither
      --cx int               Crop X
      --cy int               Crop Y
      --cw int               Crop width
      --ch int               Crop height
      --brightness float32   Brightness
      --contrast float32     Contrast (default 1)
  -h, --help                 help for video2p8
```

## License

[MIT](./LICENSE)
