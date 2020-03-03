#!/bin/bash

set -ex

npm install -g semantic-release@17.0.4 @semantic-release/exec@5.0.0

semantic-release --no-ci
