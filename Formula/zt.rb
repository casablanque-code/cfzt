class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.10.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.2/zt-darwin-arm64"
      sha256 "93d9c49d5ba2f2b8c893195c89818f3e6b0a6f63980bec4bf6f7fe1230a48623"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.2/zt-darwin-amd64"
      sha256 "80f342feb149aa98d4b350952d640981cc4e9923f45c84576b3a3530b954edc9"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.2/zt-linux-arm64"
      sha256 "5a34997dd3d556def39ec153b2404b7f91d0f1a6950f1f2a042c2b62c03a9db1"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.10.2/zt-linux-amd64"
      sha256 "e3b8058687e08b914890bb913492da8b7093111d59022feed56c66c7cf2a3b41"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
