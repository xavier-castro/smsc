class Smsc < Formula
  desc "TUI for applying package-manager release-age gates"
  homepage "https://github.com/xavier/smsc"
  url "https://github.com/xavier/smsc/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_RELEASE_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/xavier/smsc/internal/app.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/smsc"
  end

  test do
    system "#{bin}/smsc", "--version"
  end
end
