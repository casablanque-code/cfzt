class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.8.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.0/zt-darwin-arm64"
      sha256 "b14476e6dcd027fdb8fdf0d90b463e82afe0f9842ea188d1462a1a3334b2cb01"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.0/zt-darwin-amd64"
      sha256 "e8949c272e4db7ba34f0202716d154d69b5ef07ede1309b3c44ae08132964da7"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.0/zt-linux-arm64"
      sha256 "2df6004bc5e8923d16d4a51bdb1776f40cead76f9a3341462d60005c791fa23f"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.8.0/zt-linux-amd64"
      sha256 "9c723f82019b99e7165a038274d61bc7f71d9e1ab06a45d6d2d46f0a7b8c5c35"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
