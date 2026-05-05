class Skillforge < Formula
  desc "A focused CLI for managing agent skills from git repositories"
  homepage "https://github.com/rwese/skillforge"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/rwese/skillforge/releases/download/v0.1.3/skillforge_0.1.3_darwin_amd64.tar.gz"
      sha256 "4986d7bcf547c6a2b835f8959c6b0524e9e2d364a3f254683f939582375ec3fe"
    end
    on_arm do
      url "https://github.com/rwese/skillforge/releases/download/v0.1.3/skillforge_0.1.3_darwin_arm64.tar.gz"
      sha256 "b824dd465487b2155e6bce2a1265834088d8568e5e2b9a72f77114035e603b77"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/rwese/skillforge/releases/download/v0.1.3/skillforge_0.1.3_linux_amd64.tar.gz"
      sha256 "e32a05a41374041094df9b14b739fcb6f4153120d7074fd1b4efbd2684616e43"
    end
    on_arm do
      url "https://github.com/rwese/skillforge/releases/download/v0.1.3/skillforge_0.1.3_linux_arm64.tar.gz"
      sha256 "9c1bf03b5029f093d5fc5f719047d381e632d431c10ab96cd6be9b060d880428"
    end
  end

  def install
    bin.install "skillforge"
  end

  test do
    system "#{bin}/skillforge", "--help"
  end
end
