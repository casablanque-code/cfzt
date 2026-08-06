class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.8.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.3/zt-darwin-arm64"
      sha256 "18add65684f3b80719676bd5e9a1624ca869b6d141fd83bffe22653a1617cecc"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.3/zt-darwin-amd64"
      sha256 "38d8599009c7783c454077ce0ad4eaf0ffa20050120db66a52b6ae6ab878a7b2"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.3/zt-linux-arm64"
      sha256 "03e927aa7e13f7296783450b5efc5e732d2dc16999aa63728d7cd032508326d8"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.3/zt-linux-amd64"
      sha256 "31ee305f5987d4e67589e49c8f1a4f1fa647beb33b754fc816b7543f7c19b5ea"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
