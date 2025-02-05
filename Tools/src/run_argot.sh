#!/usr/bin/env bash

 ### Fixed argot version - upgrade here carefully and check that analyses are still supported
 REQUIRED_ARGOT_VERSION="v0.4.2-alpha"


 log() {
   echo "__   $1"
 }

 logok() {
    echo "OK   $1"
 }

 logfail() {
    echo "FAIL $1"
 }

 sep() {
   echo "======================================================"
 }


 argot_run_all() {
   CONFIG="$GO_SPACE/Tools/src/argot-config.yaml"
   sep
   log "Running taint"
   argot taint -config "$CONFIG"
   sep
   log "Running backtrace"
   argot backtrace -config "$CONFIG"
   sep
   log "Running syntactic"
   argot syntactic -config "$CONFIG"
 }

 ### SCRIPT LOGIC  ###

 ### Check the required argot version is present (non-zero and starts and the required full version starts with the
 # output of argot --version)
 ARGOT_VERSION=$(argot --version 2> /dev/null)
 if [[ -n $ARGOT_VERSION ]] && [[ $REQUIRED_ARGOT_VERSION == $ARGOT_VERSION* ]]; then
   logok "Argot $ARGOT_VERSION is present."
 else
   log "Installing argot ${REQUIRED_ARGOT_VERSION}"
   # Installing argot using local Go toolchain.
   # TODO: when no local toolchain, figure out how to integrate with Brazil
   go install "github.com/awslabs/ar-go-tools/cmd/argot@$REQUIRED_ARGOT_VERSION"
   ARGOT_VERSION=$(argot --version)
   logok "Successfully installed argot $ARGOT_VERSION"
 fi
mkdir -p logs/argot
argot_run_all ""
