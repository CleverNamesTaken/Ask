#!/bin/sh

#install nvim and tome

echo "[+] Installing vim"
sudo apt install vim -y

echo "[+] Installing tome and modifying .vimrc"
mkdir -p ~/.vim/pack/vendor/start
git clone https://github.com/laktak/tome.git ~/.vim/pack/vendor/start/tome

cat << EOF >> ~/.vimrc
augroup TomePlaybooks
  autocmd!
  autocmd BufNewFile,BufReadPost */.playbook/* silent! TomePlayBook
  autocmd BufNewFile,BufReadPost .playbook/*  silent! TomePlayBook
augroup END
EOF

echo "[+] Changing ask.config for tome output directory"

mkdir -p ~/.config/ask/.playbook
sed -i '/outputdir/s#:.*#: ~/.config/ask/.playbook/#' ~/.config/ask/config.yaml
