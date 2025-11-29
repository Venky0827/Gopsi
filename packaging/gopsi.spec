Name:           gopsi
Version:        0.2.0
Release:        1%{?dist}
Summary:        Gopsi - A Go-based automation tool
License:        MIT
URL:            https://github.com/Venky0827/Gopsi

# Disable automatic debug package creation
%global debug_package %{nil}

Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang

%description
Gopsi is an infrastructure automation tool written in Go.

%prep
%setup -q

%build
# Use Go Modules
export GO111MODULE=on
# Ensure static build
export CGO_ENABLED=0
# Cross-build for Linux AMD64
export GOOS=linux
export GOARCH=amd64

go build -ldflags "-s -w" -o gopsi ./cmd/gopsi

%install
install -Dm755 gopsi %{buildroot}/usr/bin/gopsi

%files
/usr/bin/gopsi

%changelog
* Fri Nov 29 2025 Venkatesh Bandaru <venky@example.com> - 0.2.0-1
- Initial RPM release of gopsi
