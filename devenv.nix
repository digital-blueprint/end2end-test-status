{
  pkgs,
  lib,
  config,
  ...
}:
{
  # More config is provided by input shared
  enterShell = ''
    echo "🛠️ End2End Test Status Dev Shell"
  '';

  # https://devenv.sh/languages/
  languages.go.enable = true;

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
  ];

  # https://devenv.sh/pre-commit-hooks/
  git-hooks.hooks = {
    golangci-lint.enable = true;
    gotest.enable = true;
    govet.enable = true;
  };

  # See full reference at https://devenv.sh/reference/options/
}
