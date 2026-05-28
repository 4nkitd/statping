class Statping < Formula
  desc "Beautiful terminal-based website monitoring tool with notifications"
  homepage "https://github.com/4nkitd/statping"
  version "1.2.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/4nkitd/statping/releases/download/v#{version}/statping-darwin-arm64.tar.gz"
      sha256 "5803ee422f31114f85476c2da8b991506f7dc44a7e488f23c0b69c2a4a03132b"
    else
      url "https://github.com/4nkitd/statping/releases/download/v#{version}/statping-darwin-amd64.tar.gz"
      sha256 "e96903c04b60d56cfed0178c58bdc096cb9b04d165eb4de1df7e693a439eb4e1"
    end
  end

  on_linux do
    odie "statping v#{version} has no prebuilt Linux binary yet — build from source: https://github.com/4nkitd/statping"
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
