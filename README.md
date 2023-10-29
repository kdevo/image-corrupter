# image-corrupter

Simple image glitcher suitable for producing nice looking locker backgrounds (swaylock, i3lock, ...).

## About this fork

At the moment, this fork is just a personalized version of the original with minimal code changes.

## Getting Started

```shell
$ git clone https://github.com/kdevo/image-corrupter
$ cd image-corrupter && go build
$ image-corrupter wallpaper.png out.png && xdg-open out.png
```

Alternatively, you can use `go install github.com/kdevo/image-corrupter` to install. Then, the binary will be at `$GOPATH/bin/image-corrupter`.

At the moment, you can only pass and output `.png` images. But that's enough to work well with `grim` + `swaylock`, `scrot` + `i3lock`.

### Screen locking example usage (`swaylock` + `grim`)

As `image-corrupter` only glitches the image for a cool background, you'd have to set up a lock script.

Example screenshot lock script for Wayland:
```bash
#!/usr/bin/env bash
image=$(mktemp)
grim -t png -l 0 -s 0.75 - | image-corrupter -mag 4 -boffset 2 - > "${image}"
trap 'rm -f -- "$image"' EXIT
swaylock -i "${image}"
```

The script above takes a screenshot with `grim` passes it through the pipe to `image-corrupter` for distorting it, and then locks the screen using `swaylock`.
The image is saved in a temporary file that is removed again after swaylock is done using it.

### Less distorted image

Default config is pretty heavy-handed. To get less disrupted images you may want to reduce blur and distortion:

```shell
$ image-corrupter -mag 1 -boffset 2 wallpaper.png out.png && xdg-open out.png
```

## Examples

Images using the default parameters:

![demo1](https://raw.githubusercontent.com/r00tman/corrupter/master/shots/example-after.png)
![demo2](https://raw.githubusercontent.com/r00tman/corrupter/master/shots/light-theme-example.png)
![demo3](https://raw.githubusercontent.com/r00tman/corrupter/master/shots/dark-theme-example.png)

With custom parameters: \
Before:
![demo4](https://raw.githubusercontent.com/r00tman/corrupter/master/shots/ps2-example-before.png)

After (custom parameters and ImageMagick dim):
![demo5](https://raw.githubusercontent.com/r00tman/corrupter/master/shots/ps2-example-after.png)
