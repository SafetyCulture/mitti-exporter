<script>
    export let label = '';
    export let onClick
    export let clazz = '';
    export let error = false
    export let type = 'active-purple'
    export let active = true;

    function actionFn() {
        if (active === false) {
            return
        }
        onClick();
    }
</script>

<button class="button {clazz}"
        class:button-error={error}
        class:button-purple={type === 'active-purple'}
        class:button-white={type === 'active-white'}
        class:button-red={type === 'active-red'}
        class:button-disabled={active === false}
        class:button-active={active === true}
        on:click={actionFn}
>{label}</button>

<style>
    .button {
        padding: 10px 16px;
        border-width: 1px;
        border-radius: 8px;
        font-weight: 500;
        font-size: 1rem;
        line-height: 1.5rem;
    }

    .button-white {
        background: #FFFFFF;
        color: #6E68DA;
        border-color: #BFC5D4;
        border-style: solid;
    }

    .button-purple {
        background: #4A42C8;
        color: #FFFFFF;
        border-color: #FFFFFF;
        border-style: solid;
    }

    .button-red {
        background: #B02010;
        color: #FFFFFF;
        border-color: #FFFFFF;
        border-style: solid;
    }

    .button-error {
        background-color: #4A42C8;
        color: #FFFFFF;
        border-color: #FFFFFF;
        border-style: solid;
    }

    .button-active:hover {
        cursor: pointer;
    }

    .button-disabled {
        background-color: #F8F9FC;
        color: #97A3B5;
        cursor: wait;
    }
</style>
