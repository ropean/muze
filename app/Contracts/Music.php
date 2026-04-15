<?php

declare(strict_types=1);

/**
 * Copyright (c) 2019-2026 guanguans<ityaozm@gmail.com>
 *
 * For the full copyright and license information, please view
 * the LICENSE file that was distributed with this source code.
 *
 * @see https://github.com/guanguans/music-dl
 */

namespace App\Contracts;

use Illuminate\Support\Collection;

interface Music
{
    /**
     * @param array<string, mixed> $options
     *
     * @return Collection<int, array<string, mixed>>
     */
    public function search(string $keyword, array $options): Collection;

    public function download(string $url, string $savedPath): void;

    /**
     * Download a song with automatic URL refresh on 403.
     *
     * Tries the URL already present in $song['url']. If the server returns 403
     * (URL expired), fetches a fresh URL via Meting and retries once.
     * Any other HTTP error is re-thrown so the caller can handle/skip it.
     *
     * @param array<string, mixed> $song     Song array as returned by search() — must contain url, url_id, source
     * @param string               $savedPath Local file path to write the audio to
     */
    public function downloadSong(array $song, string $savedPath): void;
}
