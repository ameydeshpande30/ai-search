# Neuron Search

<div align="center">

A powerful desktop application for AI-powered search, built with modern web technologies.

[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?style=flat&logo=go)](https://golang.org/dl/)
[![Node Version](https://img.shields.io/badge/Node-16+-339933?style=flat&logo=node.js)](https://nodejs.org/)
[![Wails](https://img.shields.io/badge/Wails-v2-blue?style=flat&logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjZmZmIiBzdHJva2Utd2lkdGg9IjIiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLWxpbmVqb2luPSJyb3VuZCI+PHBhdGggZD0iTTEyIDJMMiA3bDEwIDUgMTAtNS0xMC01ek0yIDE3bDEwIDUgMTAtNU0yIDEybDEwIDUgMTAtNSIvPjwvc3ZnPg==)](https://wails.io/)

</div>

---

## ✨ Features

- 🚀 **Fast & Lightweight** - Built with Wails for native performance
- 🎨 **Modern UI** - React + TypeScript with Vite for rapid development
- 🔍 **AI-Powered Search** - Intelligent search capabilities
- 💻 **Cross-Platform** - Works on Windows, macOS, and Linux
- 🔄 **Hot Reload** - Instant feedback during development
- 🛠️ **Type-Safe** - Full TypeScript support for reliable code

## 📸 Screenshots

![Screenshot 1](screenshots/Screenshot%202026-05-17%20at%207.36.21%20PM.png)

![Screenshot 2](screenshots/Screenshot%202026-05-17%20at%207.36.36%20PM.png)

## 🚀 Getting Started

### Prerequisites

Before building this application, ensure you have the following installed:

- **Go** (1.18 or later) - [Install Go](https://golang.org/dl/)
- **Node.js** (16 or later) - [Install Node.js](https://nodejs.org/)
- **Wails CLI** - Install via:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/neuron-search.git
   cd neuron-search
   ```

2. **Install frontend dependencies**
   ```bash
   cd frontend
   npm install
   cd ..
   ```

3. **Run in development mode**
   ```bash
   wails dev
   ```

## 🏗️ Building

### Production Build

To build a redistributable production package:

```bash
wails build
```

This creates an optimized executable in the `build` directory.

### Build Options

Customize your build by editing `wails.json`:

```bash
# Clean build (removes previous build artifacts)
wails build -clean

# Build for specific platform (see wails.json for options)
wails build -platform darwin/amd64
```

For detailed configuration options, see the [Wails project configuration documentation](https://wails.io/docs/reference/project-config).

## 📁 Project Structure

```
neuron-search/
├── frontend/          # React + TypeScript frontend
│   ├── src/          # Source files
│   ├── public/       # Static assets
│   └── package.json  # Frontend dependencies
├── internal/         # Internal Go packages
├── app.go           # Main application logic
├── main.go          # Application entry point
├── wails.json       # Wails configuration
├── go.mod           # Go module definition
└── go.sum           # Go dependencies checksum
```

## 🛠️ Development

### Running in Development Mode

```bash
wails dev
```

This will:
- Start a Vite development server for fast frontend hot reload
- Launch the desktop application
- Enable browser-based debugging at http://localhost:34115
- Allow you to call Go methods from browser devtools

### Frontend Development

To work on the frontend separately:

```bash
cd frontend
npm run dev
```

### Code Style

- **Go**: Follow standard Go formatting (`gofmt`)
- **TypeScript/React**: Follow ESLint and Prettier configurations
- **Commits**: Use conventional commit messages

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Write clear, descriptive commit messages
- Add tests for new features
- Update documentation as needed
- Ensure code passes all linting checks

## 📝 Configuration

Edit `wails.json` to configure:
- Application name and version
- Build settings and output paths
- Frontend framework settings
- Platform-specific options

## 🐛 Troubleshooting

### Common Issues

**Issue**: Build fails with missing dependencies
```bash
# Solution: Install Go and Node dependencies
go mod download
cd frontend && npm install
```

**Issue**: Frontend hot reload not working
```bash
# Solution: Clear Vite cache
cd frontend
rm -rf node_modules/.vite
npm run dev
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

##  Acknowledgments

- Built with [Wails](https://wails.io/) - Cross-platform Go desktop apps
- Frontend powered by [React](https://react.dev/) and [TypeScript](https://www.typescriptlang.org/)
- Styling with [Tailwind CSS](https://tailwindcss.com/)
- Icons from [Lucide](https://lucide.dev/)

## 📞 Support

If you have any questions or need help, please:
- Open an issue on GitHub

---

<div align="center">

[⬆ Back to Top](#neuron-search)

</div>
