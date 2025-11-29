Name:           gopsi
Version:        0.2.0
Release:        1%{?dist}
Summary:        Gopsi - A Go-based automation tool
License:        MIT
URL:            https://github.com/Venky0827/Gopsi

Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang

%description
Gopsi - Infrastructure automation tool written in Go.

%prep
%setup -q

%build
# Build static 64-bit binary
export GO111MODULE=on
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

go build -ldflags "-s -w" -o gopsi ./cmd/gopsi

%install
install -Dm755 gopsi %{buildroot}/usr/bin/gopsi

%files
/usr/bin/gopsi

%changelog
* Thu Nov 27 2025 Venky <anandbandari0@gmail.com> - 0.2.0-1
- Initial RPM build
