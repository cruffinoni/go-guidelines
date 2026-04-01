"""CLI entrypoint for go-guidelines-lint."""

from __future__ import annotations

from pathlib import Path

import click
from click.shell_completion import get_completion_class

from go_guidelines_lint.config import (
    ConfigError,
    DEFAULT_CONFIG_FILE,
    DEFAULT_EXCLUDE,
    DEFAULT_GUIDELINES_PATH,
    DEFAULT_INCLUDE,
    DEFAULT_MAX_LINE_LENGTH,
    AppConfig,
    LoggingConfig,
    load_config,
    merge_cli_overrides,
)
from go_guidelines_lint.cli_app import build_overrides, execute
from go_guidelines_lint.rules.validation import collect_known_rule_ids, validate_config_rule_ids

_CLI_PROG_NAME = "gg-lint"


def _export_completion(ctx: click.Context, param: click.Parameter, value: str | None) -> None:
    """Print a shell completion script and exit early."""

    del param
    if value is None or ctx.resilient_parsing:
        return

    shell_cls = get_completion_class(value)
    if shell_cls is None:
        raise click.BadParameter(f"Unsupported shell: {value}")

    complete_var = f"_{_CLI_PROG_NAME.replace('-', '_').upper()}_COMPLETE"
    completion = shell_cls(
        cli=ctx.command,
        ctx_args={},
        prog_name=_CLI_PROG_NAME,
        complete_var=complete_var,
    )
    click.echo(completion.source())
    ctx.exit(0)


@click.command(context_settings={"help_option_names": ["-h", "--help"]})
@click.argument("target", required=False)
@click.option(
    "--export-completion",
    type=click.Choice(["zsh", "bash", "fish"], case_sensitive=False),
    callback=_export_completion,
    expose_value=False,
    is_eager=True,
    help="Print shell completion script for zsh, bash, or fish and exit.",
)
@click.option(
    "--config",
    "config_path",
    type=click.Path(path_type=Path),
    default=None,
    show_default=f"{DEFAULT_CONFIG_FILE}",
    help="Path to TOML config (expects [tool.go_guidelines]).",
)
@click.option(
    "--guidelines",
    "guidelines_path",
    type=click.Path(path_type=Path),
    default=None,
    show_default=DEFAULT_GUIDELINES_PATH,
    help="Override path to guideline markdown file.",
)
@click.option(
    "--comments-guidelines",
    "comments_guidelines_path",
    type=click.Path(path_type=Path),
    default=None,
    show_default="sibling COMMENTS.md",
    help="Override path to comments guideline markdown file.",
)
@click.option(
    "--with-comments-guidelines",
    "enable_comments_guidelines",
    flag_value=True,
    default=None,
    help="Enable comments guideline rules.",
)
@click.option(
    "--no-comments-guidelines",
    "enable_comments_guidelines",
    flag_value=False,
    show_default=True,
    help="Disable comments guideline rules.",
)
@click.option(
    "--format",
    "output_format",
    type=click.Choice(["text", "json"], case_sensitive=False),
    default=None,
    show_default=AppConfig.format,
)
@click.option(
    "--fail-on",
    type=click.Choice(["error", "warning"], case_sensitive=False),
    default=None,
    show_default=AppConfig().fail_on,
)
@click.option(
    "--log-level",
    type=click.Choice(["DEBUG", "INFO", "WARNING", "ERROR"], case_sensitive=False),
    default=None,
    show_default=LoggingConfig().level,
)
@click.option(
    "--log-format",
    type=click.Choice(["text", "json"], case_sensitive=False),
    default=None,
    show_default=LoggingConfig().format,
)
@click.option("--include", multiple=True, show_default=str(DEFAULT_INCLUDE), help="Include glob pattern. Can be repeated.")
@click.option("--exclude", multiple=True, show_default=str(DEFAULT_EXCLUDE), help="Exclude glob pattern. Can be repeated.")
@click.option("--enable-rule", "enable_rules", multiple=True, help="Enable specific rule IDs (comma-separated or repeated).")
@click.option("--disable-rule", "disable_rules", multiple=True, help="Disable specific rule IDs (comma-separated or repeated).")
@click.option(
    "--max-line-length",
    type=int,
    default=None,
    show_default=str(DEFAULT_MAX_LINE_LENGTH),
    help="Maximum line length used by line-based guidelines.",
)
@click.option(
    "--max-workers",
    type=int,
    default=None,
    show_default=str(AppConfig().max_workers),
    help="Max worker threads for file analysis.",
)
@click.option(
    "--list-guidelines",
    "list_guidelines_mode",
    is_flag=True,
    default=False,
    show_default=True,
    help="List all guideline rules and their effective enablement status.",
)
@click.option(
    "--llm",
    type=click.Choice(["claude", "codex"], case_sensitive=False),
    default=None,
    help=(
        "Inject gg-lint usage instructions into the LLM context file "
        "(CLAUDE.md for claude, AGENTS.md for codex). Idempotent."
    ),
)
@click.option(
    "--git",
    "git_only",
    is_flag=True,
    default=False,
    help=(
        "Restrict scan to .go files changed per `git diff HEAD` "
        "(staged + unstaged vs last commit). Untracked files are excluded. "
        "Fails if not inside a git repository."
    ),
)
def main(
    target: str | None,
    config_path: Path | None,
    guidelines_path: Path | None,
    comments_guidelines_path: Path | None,
    enable_comments_guidelines: bool | None,
    output_format: str | None,
    fail_on: str | None,
    log_level: str | None,
    log_format: str | None,
    include: tuple[str, ...],
    exclude: tuple[str, ...],
    enable_rules: tuple[str, ...],
    disable_rules: tuple[str, ...],
    max_line_length: int | None,
    max_workers: int | None,
    list_guidelines_mode: bool,
    llm: str | None,
    git_only: bool,
) -> None:
    """Lint Go code against guideline sets."""

    try:
        config = load_config(config_path)
        known_rule_ids = collect_known_rule_ids()
        validate_config_rule_ids(config, known_rule_ids)
        overrides = build_overrides(
            target=target,
            guidelines_path=guidelines_path,
            comments_guidelines_path=comments_guidelines_path,
            enable_comments_guidelines=enable_comments_guidelines,
            output_format=output_format,
            fail_on=fail_on,
            log_level=log_level,
            log_format=log_format,
            include=include,
            exclude=exclude,
            enable_rules=enable_rules,
            disable_rules=disable_rules,
            max_line_length=max_line_length,
            max_workers=max_workers,
            known_rule_ids=known_rule_ids,
            llm=llm,
            git_only=git_only,
        )
        config = merge_cli_overrides(config, overrides)

        exit_code = execute(config, list_guidelines_mode=list_guidelines_mode)
        raise SystemExit(exit_code)

    except FileNotFoundError as exc:
        click.echo(f"Error: {exc}", err=True)
        raise SystemExit(2) from exc
    except ConfigError as exc:
        click.echo(f"Configuration error: {exc}", err=True)
        raise SystemExit(2) from exc
    except Exception as exc:  # noqa: BLE001
        click.echo(f"Unhandled error: {exc}", err=True)
        raise SystemExit(2) from exc


if __name__ == "__main__":
    main()
