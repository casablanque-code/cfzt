class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.7.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.7.2/zt-darwin-arm64"
      sha256 "163a89a79ec711737e18504a648b2ee6127d9ea4d00f7535b0c45206e17ca4a9"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.7.2/zt-darwin-amd64"
      sha256 "5844d87068a2fc2fb5088d1e76033e4705eb60746a8e0f5256d950a420c9257c"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.7.2/zt-linux-arm64"
      sha256 "25566473576fde0a675f058b8b92d40381f192aa3d4228049411674be56c5128"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.7.2/zt-linux-amd64"
      sha256 "bb9b200f84ec649da210b28f0722f4fd1301bad47315bdf75ef04381354bebe9"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
