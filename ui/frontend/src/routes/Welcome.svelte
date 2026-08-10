<script>
	import './common.css';

	import {push} from 'svelte-spa-router'
	import {ValidateApiKey} from "../../wailsjs/go/main/App.js"
	import {shadowConfig, templateCache} from '../lib/store.js';
	import ValidatableInput from "../components/ValidatableInput.svelte";
	import Button from "../components/Button.svelte";
	import StatusBar from "../components/StatusBar.svelte";
	import {BrowserOpenURL} from "../../wailsjs/runtime/runtime.js";


	let isValid = false;
	let buttonLabel = "Verify"
	let displayBadApiKeyErr = false
	let displayConnectionErr = false
	let displayValidationError = false
	let tries = 1
	let buttonActive = true
	let accessToken = $shadowConfig["AccessToken"]

	function openURL() {
		BrowserOpenURL("https://app.safetyculture.com/account/api-tokens")
	}

	function validate() {
		tries++
		isValid = false
		if (accessToken.length === 0) {
			displayValidationError = true
			return
		}

		function checkErr(errMsg) {
			if (errMsg === "connection error") {
				displayConnectionErr = true
				displayValidationError = true
			} else {
				displayBadApiKeyErr = true
				displayValidationError = true
			}
		}

		buttonActive = false
		ValidateApiKey(accessToken).then((result) => {
			if (result !== "") {
				buttonLabel = "Try again"
				checkErr(result);
			} else {
				displayValidationError = false

				if ($shadowConfig["AccessToken"] !== accessToken) {
					$shadowConfig["Export"]["TemplateIds"] = []
				}
				templateCache.set([]);
				$shadowConfig["AccessToken"] = accessToken
				push("/config")
			}
		}).finally(() => {
			buttonActive = true
		})
	}
</script>

<div class="welcome-page">
	<img id="welcome-page-logo" class="p-top-32" src="../images/logo.svg" alt="Mitti logo"/>
	<div class="h1">Welcome to Mitti Exporter</div>
	<img id="welcome-page-image" src="../images/welcome.png" alt="welcome"/>
	<div class="token-validation-text p-top-16">Generate an API token from your <span class="link" on:click={openURL} on:keypress={openURL}>SafetyCulture account</span>.</div>

	<div class="token-validation">
		<div class="input">
			<ValidatableInput placeholder="Enter API Token here" error={displayValidationError} bind:value={accessToken}/>
		</div>

		<div class="p-left-8">
			<Button label={buttonLabel} type="active-purple" active={buttonActive} error={displayValidationError} onClick={validate}/>
		</div>
	</div>

	{#if displayBadApiKeyErr}
		<div class="error-block">
			<div class="error-block-title">Unable to verify token</div>
			<div class="error-block-body">It looks like your token may be expired after 30 days of inactivity. Please generate a new token and try again.</div>
		</div>
	{/if}

	{#if displayConnectionErr}
		<div class="error-block">
			<div class="error-block-title">Connection error</div>
			<div class="error-block-body">It looks like you are not connected to the internet or behind a firewall. Please check your connection and try again.</div>
		</div>
	{/if}

	<section class="storage-info">
		<div class="note border-round-8">
			<div>
				<img src="../images/warning.svg" alt="alert icon">
			</div>
			<div>
				<div class="note-title">Important note</div>
				<div class="note-body">All files (apart from SQL) you export will be stored in the same place on your computer or server as the Mitti Exporter. If you want to change where your files get exported, please move the Mitti Exporter file itself to that place.</div>
			</div>
		</div>
	</section>

</div>

<StatusBar/>

<style>
	.welcome-page {
		display: flex;
		flex-direction: column;
		align-items: center;
        height: 100%;
        padding-bottom: 60px;
	}

	.welcome-page .h1 {
		font-size: 1.8rem;
	}

	#welcome-page-logo {
		width: 150px;
	}

	#welcome-page-image {
		width: 600px;
	}

	.token-validation {
		display: flex;
	}

	.token-validation .input {
		width: 360px;
	}

	div.token-validation-text {
		margin-bottom: 8px;
		color: #1D2330;
	}

	.token-validation-text {
		font-style: normal;
		font-weight: 400;
		font-size: 1rem;
		line-height: 1.5rem;
	}

	.storage-info {
		margin: 16px;
	}

	.note {
		background-color: #EEF1F7;
		padding: 16px;
		color: #3F495A;
		display: flex;
		flex-direction: row;
		gap: 16px;
		font-size: 0.9rem;
	}

	.note .note-title {
		font-weight: bold;
		padding-bottom: 8px;
	}

	.note .note-body {
		line-height: 1.5rem;
	}

	div.error-block {
		font-size: 0.8rem;
		color: #1D2330;
	}

	div.error-block .error-block-title {
		color: #B02010;
	}
</style>
