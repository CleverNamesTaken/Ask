#!/bin/sh

#install nvim and ultisnips

echo "[+] Installing nvim and python3 provider"
sudo apt install neovim python3 python3-pynvim -y

echo "[+] Installing ultisnips, assuming use of init.lua"
git clone https://github.com/SirVer/ultisnips ~/.local/share/nvim/site/pack/plugins/start/ultisnips
echo "vim.g.python3_host_prog = '"$(which python3)"'" >> ~/.config/nvim/init.lua
mkdir -p ~/.config/nvim/UltiSnips


echo "[+] Changing ask.config for ultisnips output directory"

sed -i '/outputdir/s#:.*#: ~/.config/nvim/UltiSnips/all/#' ~/.config/ask/config.yaml
