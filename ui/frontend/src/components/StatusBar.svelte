<script>
    import {
        OpenDirectory,
        GetSettingDir
    } from "../../wailsjs/go/main/App.js"

    import {latestVersion} from '../lib/store.js';

    import {BrowserOpenURL} from "../../wailsjs/runtime/runtime.js";
    import {isNullOrEmptyObject} from "../lib/utils.js";

    async function openFolderDialog() {
        OpenDirectory(await GetSettingDir())
    }

    function openURL(url) {
        BrowserOpenURL(url)
    }


    let currentYear = new Date().getFullYear();

</script>

<div class="bar">
    <div>
        {#if !isNullOrEmptyObject($latestVersion)}
            <span>Current version: {$latestVersion['current']}</span>
            {#if $latestVersion['current'] !== $latestVersion['latest'] && $latestVersion['latest'] !== ''}
                {#if $latestVersion['download_url'] !== ''}
                <span class="accent m-left-16 block-link" on:click={openURL($latestVersion['download_url'])} on:keydown={openURL($latestVersion['download_url'])}>Latest version available: {$latestVersion['latest']}</span>
                {:else}
                <span class="m-left-16">Latest version: {$latestVersion['latest']}</span>
                {/if}
            {/if}
        {/if}
    </div>
    <div>
        <span class="accent block-link" on:click={openFolderDialog} on:keypress={openFolderDialog}>Open logs</span>
        <span class="m-left-16 copyright">Copyright © {currentYear}</span>
    </div>
</div>

<style>
    .bar {
        position: fixed;
        padding: 14px 16px;
        background-color: #F8F9FC;
        color: #1D2330;
        display: flex;
        justify-content: space-between;
        width: 100%;
        bottom: 0;
    }

    .accent {
        color: #4A42C8;
    }

    .copyright {
        font-size: small;
    }

</style>
