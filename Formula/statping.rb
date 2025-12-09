class Statping < Formula
  desc "Beautiful terminal-based website monitoring tool with notifications"
  homepage "https://github.com/4nkitd/statping"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/4nkitd/statping/releases/download/v#{version}/statping-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_ARM64_SHA256"
    else
      url "https://github.com/4nkitd/statping/releases/download/v#{version}/statping-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_AMD64_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/4nkitd/statping/releases/download/v#{version}/statping-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    else
      url "https://github.com/4nkitd/statping/releases/download/v#{version}/statping-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
    end
  end

  def install
    bin.install Dir["statping-*"].first => "statping"
  end

  def caveats
    <<~EOS
      To start statping in the system tray:
        statping tray

      To run the TUI dashboard:
        statping start

      To enable auto-start on login:
        statping enable
    EOS
  end

  test do
    assert_match "statping", shell_output("#{bin}/statping --help")
  end
end
