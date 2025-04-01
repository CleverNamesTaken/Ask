#!/bin/bash
sudo apt install python3-venv -y
python3 -m venv /usr/local/bin/venv_ask
/usr/local/bin/venv_ask/bin/pip3 install pyyaml tabulate
cp ask.py /usr/local/bin/ask
ask --help
