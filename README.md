# Photos

A small tool for interfacing with your Anycubic Photon 3D Printer from the command line.

It is inspired by the [Universal Photon Network Controller](https://github.com/Photonsters/Universal-Photon-Network-Controller), which didn't work reliably for me on my Linux system and the code of which I considered to be _too messy_. Though the code presented herein is still considered alpha quality and thus requires some cleaning. But it works.

## Requirements

* Go
* [PHCN-UN](https://github.com/Photonsters/photon-ui-mods) to activate network functionality.

## Installation

Run `sudo make install`. (Take note that only a symbolic link is created to `/usr/local/bin/photos`.)

## Usage

```
photos connect --target IP:PORT
	Connect to the printer and save the information at ~/.photos
photos list
	List files on the plugged in USB drive
photos shell
	Opens an interactive shell. Neat for testing raw gcodes and analyzing the return messages.
photos [upload | download | delete] FILE
photos print $FILE
photos bottom-fan [off | on | during_printing]
photos top-fan [off | on | during_printing | during_led_operation]
	Fan settings are not saved, i.e. they reset when the printer is turned off and on again.
```

There are other commands, documentation is yet to be done. Look at the source.
