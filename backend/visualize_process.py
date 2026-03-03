#!/usr/bin/env python3
"""
Create an animated GIF that visualizes MBTA prediction performance over time.

Expected CSV columns:
- observation_time (required)
- predicted_arrival (required)
- actual_arrival (required)
- optional: station_id, direction, train_id

Example:
python visualize_process.py --input prediction_data.csv --output mbta_process.gif
"""

from __future__ import annotations

import argparse
from pathlib import Path

import matplotlib.pyplot as plt
import pandas as pd
from matplotlib.animation import FuncAnimation, PillowWriter


THRESHOLD_MINUTES = 3.0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate a GIF that shows prediction error and trustworthiness over time."
    )
    parser.add_argument(
        "--input",
        type=Path,
        required=True,
        help="Path to CSV with observation_time, predicted_arrival, actual_arrival.",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("mbta_process.gif"),
        help="Output GIF path (default: mbta_process.gif).",
    )
    parser.add_argument(
        "--fps",
        type=int,
        default=3,
        help="Frames per second for GIF (default: 3).",
    )
    parser.add_argument(
        "--max-frames",
        type=int,
        default=180,
        help="Cap total animation frames (default: 180).",
    )
    return parser.parse_args()


def load_data(csv_path: Path) -> pd.DataFrame:
    if not csv_path.exists():
        raise FileNotFoundError(f"Input CSV not found: {csv_path}")

    df = pd.read_csv(csv_path)
    required_cols = {"observation_time", "predicted_arrival", "actual_arrival"}
    missing = required_cols - set(df.columns)
    if missing:
        raise ValueError(f"CSV is missing required columns: {', '.join(sorted(missing))}")

    df["observation_time"] = pd.to_datetime(df["observation_time"], utc=True, errors="coerce")
    df["predicted_arrival"] = pd.to_datetime(df["predicted_arrival"], utc=True, errors="coerce")
    df["actual_arrival"] = pd.to_datetime(df["actual_arrival"], utc=True, errors="coerce")

    df = df.dropna(subset=["observation_time", "predicted_arrival", "actual_arrival"]).copy()
    if df.empty:
        raise ValueError("No valid rows after parsing datetimes.")

    df = df.sort_values("observation_time").reset_index(drop=True)
    df["error_minutes"] = (
        (df["actual_arrival"] - df["predicted_arrival"]).dt.total_seconds() / 60.0
    )
    df["abs_error_minutes"] = df["error_minutes"].abs()
    df["correct"] = (df["abs_error_minutes"] <= THRESHOLD_MINUTES).astype(int)
    df["trustworthiness_pct"] = df["correct"].expanding().mean() * 100.0
    return df


def build_animation(df: pd.DataFrame, output_path: Path, fps: int, max_frames: int) -> None:
    n_total = len(df)
    if n_total > max_frames:
        step = max(1, n_total // max_frames)
        frame_indices = list(range(0, n_total, step))
        if frame_indices[-1] != n_total - 1:
            frame_indices.append(n_total - 1)
    else:
        frame_indices = list(range(n_total))

    fig, (ax_err, ax_score) = plt.subplots(2, 1, figsize=(10, 7), constrained_layout=True)
    fig.suptitle("MBTA Prediction Process: Error and Trustworthiness", fontsize=14, fontweight="bold")

    x_all = df["observation_time"]
    y_err = df["error_minutes"]
    y_score = df["trustworthiness_pct"]

    y_err_bound = max(4.0, float(df["abs_error_minutes"].max() + 1.0))

    ax_err.axhline(0, color="gray", linewidth=1, linestyle="--")
    ax_err.axhline(THRESHOLD_MINUTES, color="#2ca02c", linewidth=1, linestyle=":")
    ax_err.axhline(-THRESHOLD_MINUTES, color="#2ca02c", linewidth=1, linestyle=":")
    ax_err.set_ylim(-y_err_bound, y_err_bound)
    ax_err.set_ylabel("Prediction Error (min)")
    ax_err.set_title("Actual - Predicted Arrival Time")
    ax_err.grid(alpha=0.25)

    ax_score.set_ylim(0, 100)
    ax_score.set_ylabel("Trustworthiness (%)")
    ax_score.set_xlabel("Observation Time")
    ax_score.set_title("Running Trustworthiness (|error| <= 3 min)")
    ax_score.grid(alpha=0.25)

    err_line, = ax_err.plot([], [], color="#1f77b4", linewidth=1.8, label="error (min)")
    err_scatter = ax_err.scatter([], [], s=20, c=[], cmap="coolwarm", vmin=-y_err_bound, vmax=y_err_bound)
    score_line, = ax_score.plot([], [], color="#2ca02c", linewidth=2.0, label="trustworthiness")
    score_text = ax_score.text(0.02, 0.92, "", transform=ax_score.transAxes, fontsize=11)

    ax_err.legend(loc="upper left")
    ax_score.legend(loc="lower right")

    def update(frame_i: int):
        end_idx = frame_indices[frame_i] + 1
        x = x_all.iloc[:end_idx]
        e = y_err.iloc[:end_idx]
        s = y_score.iloc[:end_idx]

        err_line.set_data(x, e)
        err_scatter.set_offsets(pd.DataFrame({"x": x, "y": e}).to_numpy())
        err_scatter.set_array(e.to_numpy())
        score_line.set_data(x, s)

        ax_err.set_xlim(x_all.iloc[0], x_all.iloc[-1])
        ax_score.set_xlim(x_all.iloc[0], x_all.iloc[-1])

        latest_score = s.iloc[-1]
        latest_n = end_idx
        score_text.set_text(f"Trustworthiness: {latest_score:.1f}%   n={latest_n}")
        return err_line, err_scatter, score_line, score_text

    anim = FuncAnimation(
        fig,
        update,
        frames=len(frame_indices),
        interval=max(120, int(1000 / max(1, fps))),
        blit=False,
        repeat=False,
    )

    output_path.parent.mkdir(parents=True, exist_ok=True)
    anim.save(output_path, writer=PillowWriter(fps=fps))
    plt.close(fig)


def main() -> None:
    args = parse_args()
    df = load_data(args.input)
    build_animation(df, args.output, args.fps, args.max_frames)
    print(f"GIF created: {args.output.resolve()}")
    print(f"Rows visualized: {len(df)}")
    print(f"Rule: correct if |actual - predicted| <= {THRESHOLD_MINUTES:.0f} minutes")


if __name__ == "__main__":
    main()
