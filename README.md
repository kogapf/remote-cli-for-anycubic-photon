# Photos

A small tool for interfacing your Anycubic Photon 3D Printer from the command line.

## Requirements

* Go
* [PHCN-UN](https://github.com/Photonsters/photon-ui-mods) to activate network functionality.

## Installation

Run `sudo make install`.

## Usage

```
photos connect --target IP:PORT
	Connect with the printer and saves the information at ~/.photos
photos list
	List files on the plugged in USB drive
photos upload FILE
photos download FILE
photos delete FILE
```

There are other commands, documentation is yet to be do. Look at the source.
