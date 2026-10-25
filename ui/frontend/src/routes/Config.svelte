<script>
	import './common.css';
	import dayjs from 'dayjs';
	import utc from 'dayjs/plugin/utc';
	import timezone from 'dayjs/plugin/timezone';
	import Select from 'svelte-select';
	import {
		CheckDBConnection,
		ExportCSV,
		ExportReports,
		ExportSQL,
		ExportSQLite,
		ReadBuild,
		SaveSettings,
		SelectDirectory, ValidateExportDirectory
	} from "../../wailsjs/go/main/App.js"
	import {exportConfig, shadowConfig} from '../lib/store.js';
	import {allTables} from "../lib/utils.js";
	import {Quit} from "../../wailsjs/runtime/runtime.js";
	import {push} from "svelte-spa-router";
	import FormTextInput from "../components/FormTextInput.svelte";
	import Button from "../components/Button.svelte";
	import Overlay from "../components/Overlay.svelte";
	import StatusBar from "../components/StatusBar.svelte";
	import FormPassword from "../components/FormPassword.svelte";
	import FormNumberInput from "../components/FormNumberInput.svelte";
	import ButtonSelector from "../components/ButtonSelector.svelte";
	import {DateInput} from "../components/date-picker-svelte/index.js";
	import FolderPicker from "../components/FolderPicker.svelte";

	let build = ""
	ReadBuild().then(it => {
		build = it
	})

	const statusItems = [
		{value: "true", label: "Completed only"},
		{value: "false", label: "Incomplete only"},
		{value: "both", label: "Both - completed and incomplete"}
	];
	let selectedStatus = $shadowConfig["Export"]["Inspection"]["Completed"]

	const archivedItems = [
		{value: "true", label: "Archived only"},
		{value: "false", label: "Active only"},
		{value: "both", label: "Both - active and archived"}
	];
	let selectedArchived = $shadowConfig["Export"]["Inspection"]["Archived"]

	const dataExportFormatItems = [
		{value: "csv", label: "CSV"},
		{value: "mysql", label: "MySQL"},
		{value: "postgres", label: "Postgres"},
		{value: "sqlserver", label: "SQL Server"},
		{value: "sqlite", label: "SQLite"},
		{value: "reports", label: "Reports"},
	];

	const POSTGRES_DIALECT = 'postgres';
	const SQLSERVER_DIALECT = 'sqlserver';
	const MYSQL_DIALECT = 'mysql';

	const connectionStrings = {
		postgres: 'postgresql://${dbUser}:${dbPassword}@${dbHost}:${dbPort}/${dbName}',
		sqlserver: 'sqlserver://${dbUser}:${dbPassword}@${dbHost}:${dbPort}?database=${dbName}',
		mysql: '${dbUser}:${dbPassword}@tcp(${dbHost}:${dbPort})/${dbName}?charset=utf8mb4&parseTime=True&loc=Local',
	};
	const dialects = {
		postgres: POSTGRES_DIALECT,
		sqlserver: SQLSERVER_DIALECT,
		mysql: MYSQL_DIALECT,
	};

	let dbHost = '', dbHostShowError = false, dbHostErrMsg = 'Host cannot be empty'
	let dbPort = '', dbPortPlaceholder = "e.g. " + getDefaultSQLPort($shadowConfig['Db']['Dialect']), dbPortShowError = false, dbPortErrMsg = 'Please enter a valid port number'
	let dbUser='', dbUserShowError = false, dbUserErrMsg = 'Username cannot be empty'
	let dbPassword='', dbPasswordShowError = false, dbPasswordErrMsg = 'Password cannot be empty'
	let dbName='', dbNameShowError = false, dbNameErrMsg = 'Name cannot be empty'
	let formError = false
	let dbError = false
	let showBanner = false
	let exportLocationError = false

	let selectedExportFormat = $shadowConfig["Session"]["ExportType"];

	const reportFormatItems = [
		{value: "PDF", label: "PDF"},
		{value: "WORD", label: "Word"},
		{value: "BOTH", label: "Both PDF and Word"},
	];
	let selectedReportFormat = readReportFormat()

	const timezoneItems = [
		{value: "UTC", label: "UTC"}
	];
	let selectedTimeZone = $shadowConfig["Export"]["TimeZone"]

	function readReportFormat() {
		if ($shadowConfig["Report"]["Format"] === ["PDF"]) {
			return "PDF"
		}
		if ($shadowConfig["Report"]["Format"] === ["WORD"]) {
			return "WORD"
		}
		if ($shadowConfig["Report"]["Format"].includes("PDF") && $shadowConfig["Report"]["Format"].includes("WORD")) {
			return "BOTH"
		}
		return "PDF"
	}

	function prepareReportFormatForSave() {
		switch (selectedReportFormat.value) {
			case "PDF":
				return ["PDF"]
			case "WORD":
				return ["WORD"]
			case "BOTH":
				return ["PDF", "WORD"]
			default:
				return ["PDF"]
		}
	}

	// DATE PICKER
	dayjs.extend(utc);
	dayjs.extend(timezone);
	const minDate = dayjs().add(-1, 'year').toDate()
	const zeroDate = "0001-01-01T00:00:00Z"
	let date = convertStringToDate($shadowConfig["Export"]["ModifiedAfter"], selectedTimeZone);

	// NEW SCHEDULES FILTERS
	let scheduleEndDate = $shadowConfig["Export"]["Schedule"]["EndDate"] === zeroDate
		? null
		: dayjs($shadowConfig["Export"]["Schedule"]["EndDate"]).tz(selectedTimeZone).toDate()
	if (!$shadowConfig["Export"]["Schedule"]["Statuses"]?.length) {
		$shadowConfig["Export"]["Schedule"]["Statuses"] = ["ACTIVE", "PAUSED", "FINISHED"]
	}

	function generateTemplateName() {
		let num = $shadowConfig["Export"]["TemplateIds"].length
		switch (num) {
			case 0: return "All templates selected"
			case 1: return "1 template selected"
			default: return num + " templates selected"
		}
	}

	function generateDataSetName() {
		let num = $shadowConfig["Export"]["Tables"].length
		switch (num) {
			case 0: return "All data sets selected"
			case 1: return "1 data set selected"
			default: return num + " data sets selected"
		}
	}

	function handleExportFormatUpdate(event) {
		selectedExportFormat = event.detail.value;
		if (['mysql', 'postgres', 'sqlserver'].includes(selectedExportFormat)) {
			dbPortPlaceholder = "e.g. " + getDefaultSQLPort(selectedExportFormat)
			dbPort = getDefaultSQLPort(selectedExportFormat)
		}
	}

	function setConnString() {
		if (selectedExportFormat !== '') {
			const connectionString = connectionStrings[selectedExportFormat.value];
			$shadowConfig['Db']['ConnectionString'] = connectionString.replace(/\${dbUser}/g, dbUser)
					.replace(/\${dbPassword}/g, dbPassword)
					.replace(/\${dbHost}/g, dbHost)
					.replace(/\${dbPort}/g, dbPort)
					.replace(/\${dbName}/g, dbName);
			$shadowConfig['Db']['Dialect'] = dialects[selectedExportFormat.value];
		}
	}

	function parseDbConnectionString() {
		const url = $shadowConfig["Db"]["ConnectionString"];
		function mapFields(dbStringMatch) {
			dbUser = dbStringMatch[1];
			dbPassword = dbStringMatch[2];
			dbHost = dbStringMatch[3];
			dbPort = dbStringMatch[4];
			dbName = dbStringMatch[5];
		}

		let dbStringMatch;

		// Parse SQL Server connection string
		dbStringMatch = url.match(/^sqlserver:\/\/(.+):(.+)@(.+):(\d+)\?database=(.+)$/);
		if (dbStringMatch) {
			mapFields(dbStringMatch);
			return;
		}

		// Parse MySQL connection string
		dbStringMatch = url.match(/^(.+):(.+)@tcp\((.+):(\d+)\)\/(.+)\?/);
		if (dbStringMatch) {
			mapFields(dbStringMatch);
			return;
		}

		// Parse Postgres connection string
		dbStringMatch = url.match(/^postgresql:\/\/(.+):(.+)@(.+):(\d+)\/(.+)$/);
		if (dbStringMatch) {
			mapFields(dbStringMatch);
			return;
		}

		// fail-over
		if (dbPort === '') {
			dbPort = getDefaultSQLPort($shadowConfig['Db']['Dialect'])
		}
	}

	function saveConfiguration() {
		const validFormats = {
			mysql: true,
			postgres: true,
			sqlserver: true
		};

		if (selectedExportFormat.value in validFormats) {
			setConnString();
		}

		if (selectedTimeZone.value !== '') {
			$shadowConfig["Export"]["TimeZone"] = selectedTimeZone.value
			$shadowConfig["Export"]["ModifiedAfter"] = convertDateToString(date, selectedTimeZone.value)
			$shadowConfig["Export"]["Schedule"]["EndDate"] = scheduleEndDate === null
				? zeroDate
				: dayjs(scheduleEndDate).tz(selectedTimeZone.value).endOf('day').format()
		}

		$shadowConfig["Export"]["Inspection"]["Completed"] = selectedStatus.value
		$shadowConfig["Export"]["Inspection"]["Archived"] = selectedArchived.value
		$shadowConfig["Report"]["Format"] = prepareReportFormatForSave()
		$shadowConfig["Session"]["ExportType"] = selectedExportFormat.value

		if ($shadowConfig["Export"]["Tables"].length > 0) {
			if ($shadowConfig["Export"]["Media"] === true && !$shadowConfig["Export"]["Tables"].includes("inspection_items")) {
				$shadowConfig["Export"]["Tables"].push("inspection_items")
			}

			if ($shadowConfig["Export"]["Tables"].includes("inspection_items") && !$shadowConfig["Export"]["Tables"].includes("inspections")) {
				$shadowConfig["Export"]["Tables"].push("inspections")
			}
		}


		if($shadowConfig !== {}) {
			return SaveSettings($shadowConfig)
		}
		return Promise.reject("empty configuration")
	}

	function convertDateToString(dt, tz) {
		return dayjs(dt).tz(tz).format()
	}

	function convertStringToDate(input, tz) {
		if (input === "" || input === zeroDate) {
			return minDate
		}

		return dayjs(input).tz(tz).toDate()
	}

	function getDefaultSQLPort(flavour) {
		switch (flavour) {
			case 'mysql':
				return '3306'
			case 'postgres':
				return '5432'
			case 'sqlserver':
				return '1433'
			default:
				return ''
		}
	}

	function isValidPortNumber(input) {
		return Number.isFinite(+input) && input >= 1 && input <= 65535
	}

	function validateExport() {
		let hasError = false

		switch (selectedExportFormat.value) {
			case 'csv':
				break
			case 'sqlite':
				break;
			case 'mysql':
			case 'postgres':
			case 'sqlserver':
				if (dbHost.trim() === '') {
					dbHostShowError = true
					hasError = true
				} else {
					dbHostShowError = false
				}

				if (dbPort.trim() === '' || isValidPortNumber(dbPort) === false) {
					dbPortShowError = true
					hasError = true
				} else {
					dbPortShowError = false
				}

				if (dbUser.trim() === '') {
					dbUserShowError = true
					hasError = true
				} else {
					dbUserShowError = false
				}

				if (dbPassword.trim() === '') {
					dbPasswordShowError = true
					hasError = true
				} else {
					dbPasswordShowError = false
				}

				if (dbName.trim() === '') {
					dbNameShowError = true
					hasError = true
				} else {
					dbNameShowError = false
				}
				break
			case 'reports':
				break
			default:
				hasError = true
				break
		}

		return hasError
	}

	function handleSaveAndExport() {
		showBanner = true
		formError = validateExport()
		if (formError === true) {
			return
		}

		saveConfiguration().then(_ => {
			switch (selectedExportFormat.value) {
				case 'csv':
					ValidateExportDirectory().then((result) => {
						if (result === true) {
							showBanner = false
							exportLocationError = false
							$exportConfig['items'] = getFeedsForExport()
							ExportCSV()
							push("/export/status")
						} else {
							showBanner = false
							exportLocationError = true
						}
					}).catch(() => {
						exportLocationError = true
					})
					break
				case 'mysql':
				case 'postgres':
				case 'sqlserver':
					CheckDBConnection().then(() => {
						$exportConfig['items'] = getFeedsForExport()
						showBanner = false
						ExportSQL()
						push("/export/status")
					})
					.catch(() => {
						dbError = true
					})
					break
				case 'reports':
					ValidateExportDirectory().then((result) => {
						if (result === true) {
							showBanner = false
							exportLocationError = false
							$exportConfig['items'] = ['inspections','reports']
							ExportReports()
							push("/export/status")
						} else {
							showBanner = false
							exportLocationError = true
						}
					}).catch(() => {
						exportLocationError = true
					})
					break
				case 'sqlite':
					ValidateExportDirectory().then((result) => {
						if (result === true) {
							showBanner = false
							exportLocationError = false
							$exportConfig['items'] = getFeedsForExport()
							ExportSQLite()
							push("/export/status")
						} else {
							showBanner = false
							exportLocationError = true
						}
					}).catch(() => {
						exportLocationError = true
					})
					break
			}
		}).catch(e => {
			console.debug('saveConfiguration err')
			console.debug(e)
		})
	}

	function getFeedsForExport() {
		let feedsToExport = []
		if ($shadowConfig["Export"]["Tables"] !== null && $shadowConfig["Export"]["Tables"].length > 0) {
			feedsToExport = Array.from($shadowConfig["Export"]["Tables"])
		}
		if (feedsToExport.length === 0) {
			feedsToExport = Array.from(allTables)
		}
		if ($shadowConfig["Export"]["Media"] === true && !feedsToExport.includes("media")) {
			feedsToExport.push("media")
		}
		return feedsToExport
	}

	function handleSaveAndClose() {
		saveConfiguration().then(_ => {
			Quit()
		})
	}

	function handleBackButton() {
		push("/welcome")
	}

	function handleSelectTemplates() {
		push("/config/templates")
	}

	function handleTables() {
		push("/config/datasets")
	}

	function openFolderDialog() {
		if (build === 'windows' || build === '') {
			return
		}
		SelectDirectory($shadowConfig["Export"]["Path"]).then(result => {
			if (result !== "") {
				$shadowConfig["Export"]["Path"] = result
				$shadowConfig["Export"]["MediaPath"] = result + '/media/'
			}
		})
	}

	function removeOverlay() {
		dbError = false
		showBanner = false
	}

	parseDbConnectionString();
</script>

{#if formError === false && showBanner === true}
	<Overlay>This might take a while ...</Overlay>
{/if}

{#if dbError === true}
	<Overlay>
		<div class="db-error" on:click={removeOverlay} on:keydown={removeOverlay}>
			<div>Error connecting to the database</div>
			<div>Please ensure the database details are correct</div>
			<div>Click here to go back</div>
		</div>
	</Overlay>
{/if}


<div class="config-page">
	<section class="top-nav">
		<div class="nav-left">
			<div class="block-link" on:click={handleBackButton} on:keypress={handleBackButton}>
				<img src="../images/back.svg" alt="back arrow icon">
			</div>
			<div class="h1 p-left-16">Export configuration</div>
		</div>
		<div class="nav-right">
			<Button label="Save and close" type="active-white" onClick={handleSaveAndClose}/>
			<Button label="Save and export" type="active-purple" clazz="m-left-8" error={formError} onClick={handleSaveAndExport}/>
		</div>
	</section>
	<div class="config-body m-top-8">
		<section class="filters">
			<div class="filter-title">
				<div class="h3">Filters</div>
				<div class="text-weak m-top-8">Select which sets of data you want to export from your organization.</div>
			</div>
			<ButtonSelector label="Select templates" title={generateTemplateName()} onClick={handleSelectTemplates}/>
			<ButtonSelector label="Select data sets" title={generateDataSetName()} onClick={handleTables}/>

			<div class="label">Date range from (UTC)</div>
			<div class="border-weak border-round-8 m-top-4">
				<DateInput max={new Date()} format="dd-MM-yyyy" closeOnSelection={true} bind:value={date}/>
			</div>
			{#if !$shadowConfig["Export"]["Tables"]?.length || $shadowConfig["Export"]["Tables"].includes("new_schedule_occurrences")}
				<div class="label">New schedule occurrences due up to (UTC)</div>
				<div class="border-weak border-round-8 m-top-4">
					<DateInput max={dayjs().add(5, 'year').toDate()} format="dd-MM-yyyy" placeholder="1 month from today" closeOnSelection={true} bind:value={scheduleEndDate}/>
				</div>
			{/if}
			{#if !$shadowConfig["Export"]["Tables"]?.length || $shadowConfig["Export"]["Tables"].includes("new_schedules") || $shadowConfig["Export"]["Tables"].includes("new_schedule_assignees")}
				<div class="label">Include new schedules that are:</div>
				<div class="m-top-4">
					<input type="checkbox" id="schedule-active" name="schedule-active" value="ACTIVE" bind:group={$shadowConfig["Export"]["Schedule"]["Statuses"]}>
					<label class="text-size-medium" for="schedule-active">Active</label>
					<input type="checkbox" id="schedule-paused" name="schedule-paused" value="PAUSED" bind:group={$shadowConfig["Export"]["Schedule"]["Statuses"]}>
					<label class="text-size-medium" for="schedule-paused">Paused</label>
					<input type="checkbox" id="schedule-finished" name="schedule-finished" value="FINISHED" bind:group={$shadowConfig["Export"]["Schedule"]["Statuses"]}>
					<label class="text-size-medium" for="schedule-finished">Finished</label>
				</div>
			{/if}
			<div class="label">Include completed or incomplete inspections</div>
			<div class="border-weak border-round-8 m-top-4">
				<Select items={statusItems} clearable={false} showChevron={true} searchable={false} --border="0px" bind:value={selectedStatus} >
					<div slot="chevron-icon">
						<img src="../images/arrow-down-compact.svg" alt="down arrow icon"/>
					</div>
				</Select>
			</div>

			<div class="label">Include active or archived inspections</div>
			<div class="border-weak border-round-8 m-top-4">
				<Select items={archivedItems} clearable={false} showChevron={true} searchable={false} --border="0px" bind:value={selectedArchived}>
					<div slot="chevron-icon">
						<img src="../images/arrow-down-compact.svg" alt="down arrow icon"/>
					</div>
				</Select>
			</div>
		</section>
		<section class="export-details border-round-8">
			<div class="h3">Export details</div>
			<div class="label">Export data as:</div>
			<div class="border-weak border-round-8 m-top-4">
				<Select items={dataExportFormatItems} clearable={false} showChevron={true} searchable={false} on:change={handleExportFormatUpdate} --border="0px" bind:value={selectedExportFormat}>
					<div slot="chevron-icon">
						<img src="../images/arrow-down-compact.svg" alt="down arrow icon"/>
					</div>
				</Select>
			</div>
			{#if selectedExportFormat != null && ['mysql', 'postgres', 'sqlserver'].includes(selectedExportFormat.value)}
				<div>
					<div class="label">Database details:</div>
					<FormTextInput label="Host address" error={dbHostShowError} errorMsg={dbHostErrMsg} bind:value={dbHost}/>
					<FormNumberInput label="Host port" placeholder={dbPortPlaceholder} maxlength=5 error={dbPortShowError} errorMsg={dbPortErrMsg} bind:value={dbPort} />
					<FormTextInput label="Username" error={dbUserShowError} errorMsg={dbUserErrMsg} bind:value={dbUser}/>
					<FormPassword label="Password" error={dbPasswordShowError} errorMsg={dbPasswordErrMsg} bind:value={dbPassword}/>
					<FormTextInput label="Database name" error={dbNameShowError} errorMsg={dbNameErrMsg} bind:value={dbName}/>
					<hr>
				</div>
			{/if}

			{#if selectedExportFormat != null && selectedExportFormat.value === 'reports' }
				<div class="label">Report format</div>
				<div class="border-weak border-round-8 m-top-4">
					<Select items={reportFormatItems} clearable={false} showChevron={true} searchable={false} --border="0px" bind:value={selectedReportFormat}>
						<div slot="chevron-icon">
							<img src="../images/arrow-down-compact.svg" alt="down arrow icon"/>
						</div>
					</Select>
				</div>
			{/if}

            {#if selectedExportFormat != null}
			<div class="label">Folder location</div>
			<div class="m-top-4">
				<FolderPicker label="Folder location" value={$shadowConfig["Export"]["Path"]} trimLongWords={true} onClick={build === 'windows' ? null : openFolderDialog} error={exportLocationError}/>
			</div>
            {/if}
			{#if build === 'windows'}
				<div class="sub-label m-top-4">To change folder location on Windows, move the executable to the new export folder</div>
			{/if}

			<div class="label">Export time zone</div>
			<div class="border-weak border-round-8 m-top-4">
				<Select items={timezoneItems} clearable={false} showChevron={true} searchable={false} --border="0px" bind:value={selectedTimeZone}>
					<div slot="chevron-icon">
						<img src="../images/arrow-down-compact.svg" alt="down arrow icon"/>
					</div>
				</Select>
			</div>
			<div class="label">Include:</div>
			<input type="checkbox" id="media" name="media" bind:checked={$shadowConfig["Export"]["Media"]}>
			<label class="text-size-medium" for="media">Inspection media</label>
		</section>
	</div>
</div>

<StatusBar/>

<style>
	.config-page {
		padding-top: var(--main-gutter-top);
		padding-left: var(--main-gutter-left);
		padding-right: var(--main-gutter-right);
	}
	.config-body {
		display: flex;
		justify-content: space-between;
	}

	.filters {
		width: 55%;
	}

	.export-details {
		width: 380px;
		height: 590px;
		background-color: #E9EEF6;
		padding: 16px;
		overflow-y: auto;
	}

	.db-error {
		text-align: center;
		cursor: pointer;
	}
</style>
