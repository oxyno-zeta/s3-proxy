#!/bin/bash

set -ex

npm install -g semantic-release@20.1.0 @semantic-release/exec@6.0.3

semantic-release --no-ci
