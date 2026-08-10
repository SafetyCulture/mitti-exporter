<script>
    import {latestVersion} from '../lib/store.js';
    import Button from "../components/Button.svelte";
    import {TriggerUpdate} from "../../wailsjs/go/main/App.js";
    import {push} from "svelte-spa-router";
    import {BrowserOpenURL, Quit} from "../../wailsjs/runtime/runtime.js";
    
    let updateStatus = "updating"
    let updateMessage = 'Please wait until we update your application from ' + $latestVersion['current'] + ' to ' + $latestVersion['latest']
    let cancelActive = false 
    let restartActive = false

    TriggerUpdate($latestVersion['download_url']).then(result => {
        if (result === true) {
            updateStatus = "success"
            updateMessage = 'We have updated your application to the ' + $latestVersion['latest']
            cancelActive = false
            restartActive = true
        } else {
            updateStatus = "failed"
            updateMessage = 'There was an error updating to version ' + $latestVersion['latest']
            cancelActive = true
            restartActive = false
        }
    }).catch(() => {
        updateStatus = "failed"
        cancelActive = true
        restartActive = false
    })

    function cancelHandler() {
        push("/welcome")
    }

    function restartHandler() {
        Quit()
    }
    
    function openURL(url) {
        if (url !== '') {
            BrowserOpenURL(url)
        }
    }
    
</script>
<div class="update-page">
    <img id="update-page-logo" class="p-top-32" src="../images/logo.svg" alt="Mitti logo"/>
    <div class="h1">Mitti Exporter Updater</div>
    
    <div class="middle">
        {#if updateStatus === 'updating'}
            <img class="status" src="../images/spinning.gif" alt="loading"/>
        {/if}
        {#if updateStatus === 'success'}
            <img class="status" src="../images/complete.svg" alt="ok"/>
        {/if}
        {#if updateStatus === 'failed'}
            <img class="status" src="../images/warning.svg" alt="ok"/>
        {/if}

        <div class="h3 p-top-64">{updateMessage}</div>
        {#if updateStatus === 'failed'}
            {#if $latestVersion['os'] === 'darwin'}
                <div class="p-top-8">Mitti Exporter must be moved into the Applications folder in order for the auto-update to work</div>
            {/if}   
            <div class="download-alert p-top-8" on:click={openURL($latestVersion['download_url'])} on:keydown={openURL($latestVersion['download_url'])}>
                You can manually download and install the Exporter
            </div>
        {/if}

        <div class="p-top-32">
            <Button label="Cancel" type="active-purple" active={cancelActive} onClick={cancelHandler}/>
            <Button label="Restart" type="active-purple" active={restartActive} onClick={restartHandler}/>
        </div>  
    </div>
</div>

<style>
    .update-page {
        display: flex;
        flex-direction: column;
        align-items: center;
        
        height: 100vh;
    }

    .update-page .h1 {
        font-size: 1.8rem;
    }

    #update-page-logo {
        width: 150px;
    }
    
    img.status {
        width: 200px;
        height: 200px;
    }
    
    .middle {
        flex-grow: 1;

        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
    }

    .download-alert {
        font-size: 0.9rem;
        text-align: center;
        cursor: pointer;
        color: #3060C8;
    }
</style>