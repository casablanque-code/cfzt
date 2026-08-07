class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.9.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.9.0/zt-darwin-arm64"
      sha256 "f4439cd27054f43c51bad640744074b89f0bbca5082b459e4c35d382c1a92922"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.9.0/zt-darwin-amd64"
      sha256 "cb09f7f6b6a769a18d42fbd971178aec13717b6b185f3380c7253c759d38a74f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.9.0/zt-linux-arm64"
      sha256 "ff2173c9f85aba2484cb7f9f24ae87ae5c4adae6163dccb562c941f094aba20a"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.9.0/zt-linux-amd64"
      sha256 "ab910e011376021d9807cb6f8cef6ce96a9da852c860506bec8ab56d7182afca"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
