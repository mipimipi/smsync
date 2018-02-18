# Smart Music Sync (smsync)

smsync is an easy-to-use command line application for Linux. It helps to keep huge music collections in sync and is also taking care of conversions between different formats.

smsync is made for use cases where you have a folder structure for your high quality lossless or lossy but high bit rate music that acts as a "master". From this master you replicate your music to "slaves", such as a smartphone or an SD card / hard drive for your car etc. On a smartphone or in the car you either don't have or you don't want to spend that much storage capacity that you might have for you master music storage. Thus, the replication step from the master to the slaves is not a simple copy, it's in fact a conversion step. For instance, music that is stored on the master in the lossless [FLAC format](https://xiph.org/flac/) shall be converted to [MP3](https://en.wikipedia.org/wiki/MP3) while being replicated to a slave.
Normally, you want to keep the folder structure during replication. I.e. a certain music file on the slave shall have the same folder path as its counterpart has on the master. New music is typically added to the master only. If that happened you want to update the slaves accordingly with minmal effort. If you deleted files or folders on the master for whatwever reason, also these deletions shall be propagated to the slaves. And, last but not least, as we are talking about huge music collections (several thousands or ten thousands of music files), the whole synchronization and replication process must happen in a highly automated and efficient way.

## Features

smsync takes care of all this:

### Conversion

Conversions can be configurated per slave and for each file type (i.e. for each file extension) separately. Currently, smsync supports:

* Conversions from FLAC to MP3 (using [ffmpeg](https://ffmpeg.org/))

* Conversions from MP3 to MP3 (using [lame](http://lame.sourceforge.net/))

* Simple file copy

For conversions to MP3, the quality and bitrate can be configured.

### Synchronization

The synchronization is done based on timestamps. If new music has been added to the master since the last synchronization, smsync only replicates the added files to the slave. If you have deleted files or folders on the master since the last synchronization, smsync deletes its counterparts on the slave.

The synchronization can be done stepwise. That's practical if a huge number of files has to be synchronized. In this case, the synchronization can be interrupted and continued at a later point in time.

### Parallel Processing

To make the synchronization as efficient as possible, the determination of changes since the last synchronization and the replication / conversions of files are done in parallel processes. The number of CPUs that is used for this as well as the number of parallel processe can be configured.

## Installation

### Manual Installation

smsync is written in [Golang](https://golang.org/) and thus requires the installation of Go and the [Go tool](https://golang.org/cmd/go/). Make sure that you've set the environment variable `GOPATH` accordingly. Make sure that [git](https://git-scm.com/) is installed.

To download smsync and all dependencies, open a terminal and enter

    go get github.com/mipimipi/smsync

After that, build smsync by executing

    cd $GOPATH/src/github.com/mipimipi/smsync
    make

Finally, execute

    make install

as `root` to copy the smsync binary to `/usr/bin`.

### Installation with Package Managers

For Arch Linux (and other Linux distros, that can install packages from the Arch User Repository) there's a [smsync package in AUR](https://aur.archlinux.org/packages/smsync-git/).

## Usage

### Configuration File

A slave has to have the configuration file `SMSYNC_CONF` in its root folder. This file contains the configuration for that slave in [INI format](https://en.wikipedia.org/wiki/INI_file).

Example:

    [general]
    source_dir = /home/musiclover/Music/MASTER
    num_cpus   = 4
    num_wrkrs  = 4

    [rule0]
    source    = flac
    dest      = mp3
    transform = vbr|v5|q3

    [rule1]
    source    = mp3
    transform = copy

    [rule2]
    source    = *
    transform = copy

#### General Configuration

smsync interprets the configuration file. According to the general section in the example, the root folder of the master is `/home/musiclover/Music/MASTER`. The next two entries are optional. They tell smsync to use 4 cpus and start 4 worker processes for the conversion. Per default smsync would use all available cpus and start #cpus worker processes.

#### Conversion Rules

The rules sections tell smsync what to do with the files stored in the folder structure on the master. The sections have to be named `[rule{x}]` with x = 0, 1, ... .

In the example, [rule0] tells smsync to convert FLAC files (i.e. files with the extension '.flac') to MP3, using the transformation `vbr|v5|q3`. Transformations are specified by a string that consists of three parts which are separated by '|'.

* The first part is either

  * `abr` for average bitrate,
  * `cbr` for constant bitrate
  * or `vbr` for variable bitrate

* The second part is depending of the first part:

  * for `abr` and `cbr` it must contain the desired average or constant bitrate (8, 16, 24, 32, 40, 48, 64, 80, 96, 112, 128, 160, 192, 224, 256 or 320)
  * for `vbr` it must contain the VBR quality `v{x}`, with x = 0, ..., 9.999

* The third part must contain the conversion quality `q{x}`, with x = 0, ..., 9.

Thus, the transformation string `vbr|v5|q3` tells smsync to convert with a variable bitrate, VBR quality 5 and conversion quality 3.

[rule1] tells smsync to simply copy MP3 files. If files are copied, `dest` doesn't have to be specified in the rule. Another possibility was to convert MP3 to MP3 by reducing the bitrate. This can be achieved by defining a transformation string as explained above (instead of `copy`).

[rule2] tells smsync to copy als other file, e.g. cover pictures. Without [rule2], files that do neither have the extension '.flac' nor '.mp3' would have been ignored in this example.

### Synchronization Process

For the example, let's assume the config file `SMSYNC_CONF` is stored in `/home/musiclover/Music/SLAVE`. To execute smsync for the slave open a terminal and enter

    cd /home/musiclover/Music/SLAVE
    smsync

The synchronization process is executed in the following steps:

1. smsync reads the configuration file in `/home/musiclover/Music/SLAVE`. A summary of the configuration is shown and the user is asked for confirmation.

1. smsync determines all files and directions of the master, that have changed since the last synchronization. In our example, there was no synchronization before (as otherwise the configuration file would have an entry `last_sync` that contained the time stamp of the last synchronization). Depending on the number of files, this could take a few minutes. smsync displays how many directories and files need to be synchronized and again, the user is asked for confirmation.

1. The replication / conversion of files and directories is executed. smsync shows the progress and an estimation of the remaining time.

1. After the synchronization is done, the current time is stored as `last_sync` in the configuration file.

In the example, the synchronization would replicate such a the master folder structure:

    /home/musiclover/Music/MASTER
      |- ...
      |- Rock
          |- ...
          |- Dire Straits
          |   |- Love Over Gold
          |       |- ...
          |           |- ...
          |           |- Private Investigations.flac
          |           |- ...
          |           |- cover.jpg
          |- ...
          |- Eric Clapton
              |- Unplugged
                  |- ...
                  |- Layla.mp3
                  |- ...
                  |- folder.png

to such a slave folder structure:

    /home/musiclover/Music/SLAVE
      |- ...
      |- Rock
          |- ...
          |- Dire Straits
          |   |- Love Over Gold
          |       |- ...
          |           |- ...
          |           |- Private Investigations.mp3
          |           |- ...
          |           |- cover.jpg
          |- ...
          |- Eric Clapton
              |- Unplugged
                  |- ...
                  |- Layla.mp3
                  |- ...
                  |- folder.png

### Command Line Option

smsync has only a few options:

* `--log` / `-l`
  Write a log file. The file `smsync.log` is stored in the slave folder. Independent of this option, a log file is written in case of an error.

* `--add-only` / `-a`
  Files and directories will only be added on slave side, existing files and directories will not be deleted or overwritten. This option can be used if a synchronization has been stopped before its was done.

To start a new initial synchronization, just remove the `last_sync` line from the configuration file.