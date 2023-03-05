#!/bin/bash

set -eo pipefail

script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

css=$( sass "${script_dir}/style.scss" )

echo "package main

const styleCss = \`
${css}
\`" > style.css.go

