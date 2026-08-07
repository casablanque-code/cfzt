class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v1/zt-darwin-arm64"
      sha256 "8e146a4d700c384efc5d8d5a0ca8c819953046efcc981b43dc2e741d377083b2"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v1/zt-darwin-amd64"
      sha256 "2c178f2237fe8ed54e3d9fce29bc4742b96782894b7d514201163fc39675de95"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v1/zt-linux-arm64"
      sha256 "ed0d1f008be6e979567e65e20e533cc9e82adbef4ac630db8b52a70af6048854"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v1/zt-linux-amd64"
      sha256 "d7e3f0b251656afb425e09bc293e1e460ca17df7c8ede6d42e7ce0e04c1f8fc8"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
