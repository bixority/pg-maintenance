#!/bin/bash
set -e

# Generate the configure script and Makefile
autoreconf --install --force
echo "Run './configure && make' to build the project."
