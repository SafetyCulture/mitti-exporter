<script>
    export let placeholder = ''
    export let error = false
    export let errorMsg = ''
    export let value = '';
</script>

<input
        class="form-input" class:input-error={error}
        type="text"
        placeholder={placeholder}
        bind:value={value}
/>


{#if error}
    <div class="input-error-block">
        <div class="input-error-title">{errorMsg}</div>
    </div>
{/if}

<style>
    .input-error {
        border-color: #B02010!important;
    }

    .input-error-block {
        font-size: 0.7rem;
        color: #B02010;
    }

    .input-error-title {
        color: #B02010;
    }

    .input-error::placeholder { /* Chrome, Firefox, Opera, Safari 10.1+ */
        color: #B02010;
        opacity: 1; /* Firefox */
    }

    .input-error:-ms-input-placeholder { /* Internet Explorer 10-11 */
        color: #B02010;
    }

    .input-error::-ms-input-placeholder { /* Microsoft Edge */
        color: #B02010;
    }

    .form-input {
        width: 100%;
        padding: 10px 16px;
        border-radius: 8px;
        border: 1px solid #BFC5D4;
        font-size: 1rem;
        line-height: 1.5rem;
    }
</style>
