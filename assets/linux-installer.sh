#!bin/sh

echo "installing minepkg"
mkdir -p $HOME/.minepkg
curl -L "https://storage.googleapis.com/minepkg-client/latest/minepkg-linux-amd64" -o $HOME/.minepkg/minepkg
chmod +x $HOME/.minepkg/minepkg
if [ "$(id -u)" -eq 0 ]; then
  ln -fs $HOME/.minepkg/minepkg /usr/local/bin/minepkg
else
  sudo ln -fs $HOME/.minepkg/minepkg /usr/local/bin/minepkg
fi