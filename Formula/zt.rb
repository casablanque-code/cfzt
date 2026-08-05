class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.8.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.2/zt-darwin-arm64"
      sha256 "dcf480076e5f382b750cc5c8de887b11e176e91a91ec8ca002a8a44d548a9ea0"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.2/zt-darwin-amd64"
      sha256 "7520149ff9668c02d7eef797e485a2b50027de723ef1cebb8c7dd7b0e1c089e7"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.2/zt-linux-arm64"
      sha256 "bf390db91211fff0639bf3bddfc9b46c5a45393ea0b46f8756bc333440302fd4"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.2/zt-linux-amd64"
      sha256 "43d5c2e052236faba6280c68418469f0dccd5a7b6d09df3537ca202019bf1873"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
