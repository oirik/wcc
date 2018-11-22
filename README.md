# wcc

[![Build Status](https://travis-ci.org/oirik/wcc.svg?branch=master)](https://travis-ci.org/oirik/wcc)
[![GoDoc](https://godoc.org/github.com/oirik/wcc?status.svg)](https://godoc.org/github.com/oirik/wcc)
[![apache license](https://img.shields.io/badge/license-Apache-blue.svg)](LICENSE)

WebChangeChecker (wcc) is a command-line tool which checks whether a website is changed or not, written by golang.

* Manage more then one website
* Notify to your slack incoming webhook when the websites have updated
* Could check only the part of a website page by specifing the css selector (you could get this from Chrome Development Tool)


# Install

```sh
$ go get github.com/oirik/wcc
```

Or download binaries from [github releases](https://github.com/oirik/wcc/releases)

# Usage

## Add checking website

```sh
$ wcc add {url} ({css-selector})
```

if you didn't set css-selector, the whole page is checked.

## Check website updated

```sh
$ wcc check
```

if you want to notify slack, 

```sh
$ wcc check -slack {slack-incoming-webhook-url}
```

As wcc is the `on-demand` command-line tool, if you want to check regularly, you would register this command to linux cron or windows task scheduler.

## List checking websites

```sh
$ wcc list
```

## Remove checking website

```sh
$ wcc list  # check the No. of removing
$ wcc rm {No.}
```


