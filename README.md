# Grab

<p align="center">
  <img src="https://raw.githubusercontent.com/mariuslevraidevrai/grab/main/assets/grab.png" alt="Grab Banner" width="600"/>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Golang"/></a>
  <a href="https://www.linux.org/"><img src="https://img.shields.io/badge/Linux-FCC624?style=for-the-badge&logo=linux&logoColor=black" alt="Linux"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge" alt="License: MIT"/></a>
</p>

**Grab** is a fast, lightweight, and modern command-line file downloader written in Go 🔵. It accelerates downloads by splitting files into multiple chunks and downloading them concurrently using HTTP range requests.

---

## ✨ Features

* **⚡ Fast Multi-Threaded Downloads:** Splits files into customizable chunks (`-c`) for maximum download speeds.
* **🔄 Automatic Fallback:** Gracefully switches to single-threaded downloading if the target server does not support parallel range requests (`Accept-Ranges`).
* **📊 Interactive CLI Progress Bar:** Real-time feedback displaying download speed, ETA, and progress rendered cleanly via `progressbar/v3`.
* **🧠 Smart Path Resolution:** Automatically handles target output directories and custom file naming options.
* **🪶 Lightweight Binary:** Stripped symbols and optimized compilation flag options (`-ldflags "-s -w"`) ensure a minimal footprint (~4MB).
* **🖥️ Cross-Platform Compilation:** Built-in Makefile support for 32-bit x86, 64-bit amd64, ARM, and ARM64 architectures.

---

## ⚡ Benchmark

Here is a quick performance comparison downloading a **100 MB** file (e.g., OVH test file):

| Tool | Mode / Threads | Time | Speed Improvement |
| :--- | :--- | :--- | :--- |
| **`wget`** | Single-threaded | **~12s** | *Baseline* |
| **`grab`** | 4 Chunks (`-c 4`) | **~7s** | **~41% faster** |

---

## 🛠️ Installation

### Prerequisites

* **[Go](https://go.dev/)** 1.18 or higher 🐹
* **Linux** / Unix environment (or WSL) 🐧
* **`make`** (optional, for convenience) 🛠️

### Build from Source

1. Clone the repository:
   git clone https://github.com/mariuslevraidevrai/grab.git
   cd grab

2. Build or install the binary:

   * **Standard Build (Native Architecture):**
     make build

   * **Cross-Compilation for Specific Architectures:**
     # Build 64-bit x86 binary
     make build-amd64

     # Build 32-bit x86 binary
     make build-386

     # Build ARM 64-bit binary (e.g., Raspberry Pi 64-bit, ARM servers)
     make build-arm64

     # Build ARM 32-bit binary (e.g., older Raspberry Pi)
     make build-arm

     # Build all architectures at once (output in ./build directory)
     make build-all

   * **Install Globally:**
     sudo make install

   * **Using Standard Go CLI:**
     go build -ldflags "-s -w" -o grab .

---

## 🚀 Usage

grab [options] <URL>

or explicitly specify arguments:

grab -u <URL> -o <PATH> -c <CHUNKS>

### Options & Flags

| Flag | Long Option | Description | Default |
| :--- | :--- | :--- | :--- |
| `-u` | — | Target URL to download | `""` |
| `-o` | — | Output file directory or custom destination path | `"."` |
| `-c` | — | Number of concurrent download chunks/threads | `4` |
| `-v` | `--version` | Display program version and info banner | `false` |

---

## 💡 Examples

#### Basic Download
Download a file to the current directory using 4 parallel threads:
grab https://example.com/file.zip

#### Specify Output Directory or File Name
Save the downloaded file directly to a specific folder or under a custom name:
# Save to a directory
grab -o ~/Downloads https://example.com/file.zip

# Save with a custom filename
grab -o ~/Downloads/custom_name.zip https://example.com/file.zip

#### Custom Thread Count
Speed up large file downloads by specifying 8 concurrent chunks:
grab -c 8 https://example.com/large-file.iso

---

## 🧹 Uninstallation

If installed via `make`, you can easily clean up or remove the global binary:

# Remove build artifacts and build/ directory
make clean

# Uninstall binary from /usr/local/bin
sudo make uninstall

---

## 📜 License

Distributed under the **MIT License** 📄. See `LICENSE` for more information.