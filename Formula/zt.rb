class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.10.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.0/zt-darwin-arm64"
      sha256 "10d953894c102bca69e343f70c803c550a74e6ce6f2ed431e3a9cedb7b361ef0"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.0/zt-darwin-amd64"
      sha256 "41bdc18c0fc57f22ad7207d916278abccefafa8f17c2fcddfd69abce2e2e2cfa"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.0/zt-linux-arm64"
      sha256 "67f734d8f3b5194071795361a0a2ac4f866ed1c3ca0a886ebccfec1f413935ba"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.0/zt-linux-amd64"
      sha256 "ff462986288714eed53016c121c09bb69f70c99e6299c334ba177a52250d5d2c"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
