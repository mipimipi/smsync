<!--
SPDX-FileCopyrightText: 2018-2020 Michael Picht

SPDX-License-Identifier: GPL-3.0-or-later
-->

## [Release 3.4.0](https://gitlab.com/mipimipi/smsync/-/tags/3.4.0) (2020-02-09)

### Added

* Support interruption of synchronization and later continuation ([#24](https://gitlab.com/mipimipi/smsync/issues/24))

### Changed

* Moved project to [GitLab](https://gitlab/mipimipi/smsync)
* Switched to [go-utils library](https://gitlab.com/mipimipi/go-utils)

## [Release 3.3.1](https://gitlab.com/mipimipi/smsync/-/tags/3.3.1) (2018-12-28)

### Removed

* Possibility to do synchronization stepwise (it turned out that this simply does not work)

### Fixed

* Buggy selection of files and directories fixed

## [Release 3.3.0](https://gitlab.com/mipimipi/smsync/-/tags/3.3.0) (2018-12-27)

### Added

* Rename Config.SrcDirPath to Config SrcDir (same for Config.TrgDirPath) ([#20](https://gitlab.com/mipimipi/smsync/issues/20))
* Clean filenames that are read from config ([#18](https://gitlab.com/mipimipi/smsync/issues/18))
* Enhance docu wrt. conversion modes ([#22](https://gitlab.com/mipimipi/smsync/issues/19))
* Explain what to do if the conversion scope has been extended per config file ([#17](https://gitlab.com/mipimipi/smsync/issues/17))

### Changed

* Rename smsync.err ([#19](https://gitlab.com/mipimipi/smsync/issues/19))
* Changed cli option `--initialize` to `--init`

### Fixed

* Display indication that errors occurred ([#15](https://gitlab.com/mipimipi/smsync/issues/15))

## [Release 3.2.1](https://gitlab.com/mipimipi/smsync/-/tags/3.2.1) (2018-12-26)

### Added

* Extended documentation (README.md)

### Fixed

* conversion to opus: 'vbr:128' is not a valid conversion ([#21](https://gitlab.com/mipimipi/smsync/issues/21))

## [Release 3.2.0](https://gitlab.com/mipimipi/smsync/-/tags/3.2.0) (2018-12-24)

### Added

* Store output of FFMPEG in case of error ([#11](https://gitlab.com/mipimipi/smsync/issues/11))
* Optimization of determination of new/changed directories and files
* Reorganization of code/responsibility between backend and UI
* Improvement of status display

### Fixed

* Empty folders ([#10](https://gitlab.com/mipimipi/smsync/issues/10))
* Empty log file ([#12](https://gitlab.com/mipimipi/smsync/issues/12))
* Added source directories are not recognized ([#13](https://gitlab.com/mipimipi/smsync/issues/13))
* Renaming of source directories not handled properly ([#14](https://gitlab.com/mipimipi/smsync/issues/14))

## [Release 3.1.1](https://gitlab.com/mipimipi/smsync/-/tags/3.1.1) (2018-12-20)

### Added

* Enhanced progress display by estimated target size, number of errors and average compression rate

### Fixed

* Erroneous calculation of disk space

## [Release 3.1.0](https://gitlab.com/mipimipi/smsync/-/tags/3.1.0) (2018-12-19)

### Added

* Possibility to exclude source folders from conversion ([#7](https://gitlab.com/mipimipi/smsync/issues/7))
* Indicator for insufficient diskspace on target device ([#8](https://gitlab.com/mipimipi/smsync/issues/8))

## [Release 3.0.3](https://gitlab.com/mipimipi/smsync/-/tags/3.0.3) (2018-12-17)

### Added

* Non-interactive mode ([#2](https://gitlab.com/mipimipi/smsync/issues/2))
* Display name of file that is currently worked on ([#4](https://gitlab.com/mipimipi/smsync/issues/4))
* Progress display simplified
* Possibility to continue conversion after interruption with e.g. CTRL-C
* Major redesign under the hood as preparation for a future graphical UI in addition to the command line interface

### Changed

* Configuration file has been changed from INI (`SMSYNC.CONF`) to YAML format (`symsync.yaml`) and renamed accordingly.

### Removed

* Command line option `--addonly` / `-a`. This option is obsolete, since now the processing status "work in progress" is used to trigger that system behavior.

### Fixed

* Conversion Mode Copy is invalid ([#1](https://gitlab.com/mipimipi/smsync/issues/1))
* Docu error ([#5](https://gitlab.com/mipimipi/smsync/issues/5))
