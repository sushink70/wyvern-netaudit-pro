#!/bin/bash

# Update system and install traceroute
echo "Installing system-level dependencies..."
if [ -x "$(command -v apt)" ]; then
    sudo apt update
    sudo apt install -y traceroute
elif [ -x "$(command -v yum)" ]; then
    sudo yum install -y traceroute
else
    echo "Unsupported package manager. Please install traceroute manually."
    exit 1
fi

# Install Python dependencies
echo "Installing Python dependencies..."
pip install -r requirements.txt

echo "Setup complete!"
