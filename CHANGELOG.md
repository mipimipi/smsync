# [Release 3.0.1](https://github.com/mipimipi/smsync/releases/tag/3.0.1) (2018-12-16)

## Added

* Non-interactive mode ([#2](https://github.com/mipimipi/smsync/issues/2))
* Display name of file that is currently worked on ([#4](https://github.com/mipimipi/smsync/issues/4))
* Progress display simplified
* Possibility to continue conversion after interruption with e.g. CTRL-C
* Major redesign under the hood as preparation for a future graphical UI in addition to the command line interface

## Changed

* Configuration file has been changed from INI (`SMSYNC.CONF`) to YAML format (`symsync.yaml`) and renamed accordingly.

## Removed

* Command line option `--addonly` / `-a`. This option is obsolete, since now the processing status "work in progress" is used to trigger that system behavior.

## Fixed

* Conversion Mode Copy is invalid ([#1](https://github.com/mipimipi/smsync/issues/1))
* Docu error ([#5](https://github.com/mipimipi/smsync/issues/5))