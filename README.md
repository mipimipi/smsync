# Smart Music Sync (smsync)

keeps huge music collections in sync and is taking care of conversions between different formats. It an easy-to-use command line application for Linux. 

smsync is made for use cases where you have a folder structure for your high quality lossless or lossy but high bit rate music that acts as a "master". From this master you replicate your music to "slaves", such as a smartphone or an SD card / hard drive for your car etc. On a smartphone or in the car you either don't have or you don't want to spend that much storage capacity that you might have for you master music storage. Thus, the replication step from the master to the slaves is not a simple copy, it's in fact a conversion step. For instance, music that is stored on the master in the lossless [FLAC format](https://xiph.org/flac/) shall be converted to [MP3](https://en.wikipedia.org/wiki/MP3) while being replicated to a slave.
Normally, you want to keep the folder structure during replication. I.e. a certain music file on the slave shall have the same relative folder path as its counterpart has on the master. New music is typically added to the master only. If that happened you want to update the slaves accordingly with minimal effort. If you deleted files or folders on the master for whatever reason, these deletions shall be propagated to the slaves as well. And, last not least, as we are talking about huge music collections (several thousands or ten thousands of music files), the whole synchronization and replication process must happen in a highly automated and performant way.

## Features

smsync takes care of all this:

### Conversion

Conversions can be configurated per slave and file type (i.e. for each file extension) separately. Currently, smsync supports:

* Conversions to FLAC, from [WAV](https://en.wikipedia.org/wiki/WAV) and FLAC.

* Conversions to MP3, from WAV, FLAC, MP3, [OGG (Vorbis)](https://en.wikipedia.org/wiki/Vorbis) and [OPUS](https://en.wikipedia.org/wiki/Opus_(audio_format)).

* Conversions to OGG (Vorbis), from WAV, FLAC, MP3, OGG(Vorbis) and OPUS.

* Conversions to OPUS, from WAV, FLAC, MP3, OGG(Vorbis) and OPUS.

For all these conversions, [ffmpeg](https://ffmpeg.org/) is used. In addition, a simple file copy without any format conversion is supported as well.

### Synchronization

The synchronization between master and slave is done based on timestamps. If new music has been added to the master since the last synchronization, smsync only replicates / converts the added files. If you have deleted files or folders on the master since the last synchronization, smsync deletes its counterparts on the slave.

The synchronization can be done stepwise. That's practical if a huge number of files has to be synchronized. In this case, the synchronization can be interrupted and continued at a later point in time.

### Parallel Processing

To make the synchronization as efficient as possible, the determination of changes since the last synchronization and the replication / conversion of files are done in parallel processes. The number of CPUs that is used for this as well as the number of parallel processes can be configured.

## Installation

### Manual Installation

smsync is written in [Golang](https://golang.org/) and thus requires the installation of Go and the [Go tool](https://golang.org/cmd/go/). Make sure that you've set the environment variable `GOPATH` accordingly, and make also sure that [git](https://git-scm.com/) is installed.

To download smsync and all dependencies, open a terminal and enter

    $ go get github.com/mipimipi/smsync

After that, build smsync by executing

    $ cd $GOPATH/src/github.com/mipimipi/smsync
    $ make

Finally, execute

    $ make install

as `root` to copy the smsync binary to `/usr/bin`.

### Installation with Package Managers

For Arch Linux (and other Linux distros, that can install packages from the Arch User Repository) there's a [smsync package in AUR](https://aur.archlinux.org/packages/smsync-git/).

## Usage

### Configuration File

A slave has to have a configuration file with the name `SMSYNC.CONF` in its root folder. This file contains the configuration for that slave in [INI format](https://en.wikipedia.org/wiki/INI_file).

Example:

    [general]
    source_dir = /home/musiclover/Music/MASTER
    num_cpus   = 4
    num_wrkrs  = 4

    [rule0]
    source = flac
    target = mp3
    conversion = abr:192|cl:3

    [rule1]
    source = mp3
    conversion = copy

    [rule2]
    source = *
    conversion = copy

#### General Configuration

smsync interprets the configuration file. According to the general section in the example, the root folder of the master is `/home/musiclover/Music/MASTER`. The next two entries are optional. They tell smsync to use 4 cpus and start 4 worker processes for the conversion. Per default smsync uses all available cpus and starts #cpus worker processes.

#### Conversion Rules

The rules sections tell smsync what to do with the files stored in the folder structure on the master. The sections have to be named `[rule{x}]` with x = 0, 1, ... .

In the example, `[rule0]` tells smsync to convert FLAC files (i.e. files with the extension '.flac') to MP3, using the conversion `vbr:5|cl:3`. These conversions rules are strings that consist of different parts which are separated by '|'. The supported content of conversion rules depends on the target format - see detailed explanation below.

`[rule1]` of the example tells smsync to simply copy MP3 files. If files are copied, `target` doesn't have to be specified in the rule. Another possibility was to convert MP3 to MP3 by reducing the bit rate. This can be achieved by defining a dedicated conversion rule as explained above (instead of `copy`).

`[rule2]` tells smsync to copy als other files, e.g. cover pictures. Without `[rule2]`, files that do neither have the extension '.flac' nor '.mp3' would have been ignored in this example.

#### Format-dependent conversion parameters

Basically,to things can be determined with a conversion string:

1. The target bit rate.

    Here, it's often distinguished between

    * a constant bit rate (CBR), where the bit rate is constant - a special case is the "hard constant bitrate" (HCBR), which is specific to the OPUS format and guarantees that all frames have the same size,
    * an average bit rate (ABR), where the bit rate of the file is varies, but in average it reaches a certain value,
    * or a variable bit rate (VBR), where the bit rate also varies, but the compression is done according to a certain quality.

1. The compression quality

    Many, but not all, target formats support a "compression level" (CL). With this parameter, the compression quality can be steered.

The available or supported conversion parameters depend on the target format. The following sections describe the different possibilities.

##### MP3

MP3 supports ABR, CBR, both with bit rates from 8 to 500 kbps (kilo bit per second), and VBR with a quality from 0 to 9 (where 0 means highest quality). In addition, MP3 supports a compression level (CL), which can have values 0, ..., 9 where 0 means the highest quality. Thus, the conversion `abr:192|cl:3` in the example above specifies an average bit rate of 192 kbps and a compression level of 3.

See also: [FFMpeg Codec Documentation](http://ffmpeg.org/ffmpeg-codecs.html#libmp3lame-1)

##### FLAC

FLAC only supports a compression level (parameter `cl`). Possible values are: 0, ..., 12 where 0 means the highest quality. 5 is the default. Thus, for a conversion to FLAC, if no conversion rule is specified in `SMSYNC.CONF`, `cl:5` is assumed. 

See also: [FFMpeg Codec Documentation](http://ffmpeg.org/ffmpeg-codecs.html#flac-2)

##### OGG (Vorbis)

This format supports conversions with average and variable bit rate. For AVR, bit rates from 8 to 500 kbps are supported. For VBR, possible values are -1.0, ..., 10.0 where 10.0 means the best quality. VBR with quality 3.0 is the default. Thus, for a conversion to OGG (Vorbis), if no conversion rule is specified in `SMSYNC.CONF`, `vbr:3.0` is assumed. OGG (Vorbis) doesn't support compression levels.

See also: [FFMpeg Codec Documentation](http://ffmpeg.org/ffmpeg-codecs.html#libvorbis)

##### OPUS

OPUS supports conversions with average, constant and hard constant bit rate. The latter guarantees that all frames have the same size. Allowed values are 6 to 510 kbps. In addition, OPUS supports a compression level that ranges from 0 to 10, where 10 is the highest quality. If no compression level is specified, `cl:10`is assumed.

See also: [FFMpeg Codec Documentation](http://ffmpeg.org/ffmpeg-codecs.html#libopus-1)

### Synchronization Process

Coming back to the example above. Let's assume the config file `SMSYNC.CONF` is stored in `/home/musiclover/Music/SLAVE`. To execute smsync for the slave open a terminal and enter

    $ cd /home/musiclover/Music/SLAVE
    $ smsync

The synchronization process is executed in the following steps:

1. smsync reads the configuration file in `/home/musiclover/Music/SLAVE`. A summary of the configuration is shown and the user is asked for confirmation.

1. smsync determines all files and directories of the master, that have changed since the last synchronization. In our example, there was no synchronization before (as otherwise the configuration file would have an entry `last_sync` that contained the time stamp of the last synchronization). Depending on the number of files, this could take a few minutes. smsync displays how many directories and files need to be synchronized and again, the user is asked for confirmation.

1. The replication / conversion of files and directories is executed. smsync shows the progress and an estimation of the remaining time.

1. After the synchronization is done, the current time is stored as `last_sync` in the configuration file.

In the example, the synchronization would convert such a master folder structure:

    /home/musiclover/Music/MASTER
      |- ...
      |- Rock
          |- ...
          |- Dire Straits
          |   |- ...
          |   |- Love Over Gold
          |        |- ...
          |        |- Private Investigations.flac
          |        |- ...
          |        |- cover.jpg
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
          |   |- ...
          |   |- Love Over Gold
          |        |- ...
          |        |- Private Investigations.mp3
          |        |- ...
          |        |- cover.jpg
          |- ...
          |- Eric Clapton
              |- Unplugged
                  |- ...
                  |- Layla.mp3
                  |- ...
                  |- folder.png

### Command Line Options

smsync has only a few options:

* `--log` / `-l`
  Write a log file. The file `smsync.log` is stored in the root folder of the slave. A log file is always written in case of an error.

* `--add-only` / `-a`
  Files and directories will only be *added* on slave side, existing files and directories will not be deleted or overwritten. This option can be used if a synchronization has been stopped before it was completely done.

To start a new initial synchronization, just remove the `last_sync` line from the configuration file.