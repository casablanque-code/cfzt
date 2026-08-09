class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.10.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.1/zt-darwin-arm64"
      sha256 "ec9b62c308540d86d6bd2c6001a313d0140f41a9e977e9c566f52ba516400caa"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.1/zt-darwin-amd64"
      sha256 "d129bd8b2bd4bb739588cd8ecc812256cd0a9e5ff0527607247b09a3cf03287c"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.1/zt-linux-arm64"
      sha256 "352e7f6f0930d49e0cd12d0023e8116ade2705191cf78de722fa709ad3ab68ec"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.1/zt-linux-amd64"
      sha256 "13008fbde41edfd4623c9d7a1c1bf38a9c401125ad1b3a4872e246f7da9b201a"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
