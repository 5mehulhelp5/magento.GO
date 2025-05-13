sudo rm -rf /usr/local/go
wget https://go.dev/dl/go1.24.3.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.24.3.linux-arm64.tar.gz

# Add Go to PATH
if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
  echo 'export PATH="/usr/local/go/bin:$PATH"' >> ~/.bashrc
fi
source ~/.bashrc

go version

