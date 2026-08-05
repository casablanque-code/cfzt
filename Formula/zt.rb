class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.8.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.1/zt-darwin-arm64"
      sha256 "ec72953e4bd62abf04decccc3e8280fd7930f762c2b0eb2f181c8f60e348d18b"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.1/zt-darwin-amd64"
      sha256 "d482d1b3829249c215a00acea9efb8050d0a75420cd59a38c21aab10d4be0d01"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.1/zt-linux-arm64"
      sha256 "a12b44d56cef56cae162e70b57bc88b3b910d8b64ee02102c1d8decde787e730"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.1/zt-linux-amd64"
      sha256 "82a3399b24010336efea7f7336b634b28d6b289e83af5cfa06f80290f57728ba"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
