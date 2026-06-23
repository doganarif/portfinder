# pf

A modern CLI tool to identify and manage processes using network ports.  
Built with Go for speed, clarity, and zero runtime dependencies.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)

## 🎬 Preview

![pf in action](pf.gif)

---

## ✨ Features

- 🔍 **Smart Process Detection** — Instantly find what's using your ports
- 📁 **Project Awareness** — Shows which project/directory owns the process
- 🐳 **Docker Support** — Identifies containerized processes
- 🎯 **Quick Actions** — Kill processes interactively or directly
- 📊 **Port Overview** — Check all common development ports
- 🚀 **Fast & Lightweight** — Single binary, no runtime dependencies

---

## 📦 Installation

### ✅ Using Homebrew (macOS/Linux)

```bash
brew tap doganarif/tap
brew install pf
```

### 🪟 Windows

Download the latest `.exe` file from the [Releases Page](https://github.com/doganarif/portfinder/releases):

- For 64-bit Windows: `pf-windows-amd64.exe`
- For ARM64 Windows: `pf-windows-arm64.exe`

Rename it to `pf.exe` and add to your PATH.

### 🧰 Using Go

```bash
go install github.com/doganarif/portfinder/cmd/portfinder@latest
```

This installs the binary as `pf`.

### 📁 Download Binary

Grab the latest release from the [Releases Page](https://github.com/doganarif/portfinder/releases).

---

## 🧪 Usage

### 🔍 Check a specific port

```bash
pf 3000
```

Output:

```
🔍 Port 3000 is in use by:

Process     node
PID         48291
Command     npm run dev
Project     ~/projects/my-react-app
Started     3 hours ago

Kill this process? [y/n]
```

---

### 📊 Check common development ports

```bash
pf check
```

Example output:

```
📊 Common Development Ports:

Frontend:
  ❌ 3000: node (my-react-app)
  ✅ 3001: free
  ✅ 4200: free
  ❌ 5173: vite (my-vue-app)
  ✅ 8080: free

Backend:
  ✅ 4000: free
  ❌ 5000: python (flask-api)
  ✅ 8000: free
  ✅ 9000: free

Databases:
  ✅ 3306: free
  ❌ 5432: postgres (docker)
  ❌ 6379: redis
  ✅ 27017: free
```

---

### 📋 List all ports in use

```bash
pf list
```

---

### 💀 Kill a process

```bash
pf kill 3000
```

---

## ⚙️ Common Ports Reference

| Port  | Common Use                |
| ----- | ------------------------- |
| 3000  | React, Node.js, Rails     |
| 3001  | Create React App fallback |
| 4200  | Angular                   |
| 5173  | Vite                      |
| 5000  | Flask, Python servers     |
| 8000  | Django                    |
| 8080  | General web development   |
| 3306  | MySQL/MariaDB             |
| 5432  | PostgreSQL                |
| 6379  | Redis                     |
| 27017 | MongoDB                   |
| 9200  | Elasticsearch             |
| 9090  | Prometheus                |
| 3100  | Grafana Loki              |
| 8983  | Solr                      |

---

## 🛠️ Configuration

You can override the default list of common ports by creating a config file at:

```bash
~/.config/portfinder/config.json
```

Example:

```json
{
  "common_ports": [3000, 3001, 5173, 5000, 8000]
}
```

---

## 🧑‍💻 Development

### Prerequisites

- Go 1.21+
- Make (optional)

### Building from source

```bash
# Clone the repository
git clone https://github.com/doganarif/portfinder.git
cd portfinder

# Build
make build

# Run tests
make test

# Install locally
make install
```

---

## 📁 Project Structure

```
pf/
├── cmd/
│   └── portfinder/     # CLI entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── process/        # Process detection logic
│   └── ui/             # Terminal UI components
├── Makefile            # Build automation
└── README.md           # This file
```

---

## 🤝 Contributing

Contributions are welcome! Please open an issue or pull request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📜 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file.

---

## 🙌 Acknowledgments

- Inspired by the frustration of "port already in use" errors
- Built using [Cobra](https://github.com/spf13/cobra) for CLI
- Terminal UI powered by [Bubbletea](https://github.com/charmbracelet/bubbletea)

---

## 🧑 Author

**Arif Doğan**

- GitHub: [@doganarif](https://github.com/doganarif)
- Twitter: [@arifcodes](https://twitter.com/arifcodes)

> If you find this tool useful, please consider giving it a ⭐️ on GitHub!

<!-- arif-signature:start -->

---

Built by [Arif Dogan](https://arif.sh) - production AI and backend engineer.

I help SaaS teams ship production AI features, fast backends, and reliable developer tools.

[Work with me](https://arif.sh/work) | [Book a 30-min intro](https://calendar.superhuman.com/book/11SzDRA4zo8tuYoehO/A3kIl)

<!-- arif-signature:end -->
