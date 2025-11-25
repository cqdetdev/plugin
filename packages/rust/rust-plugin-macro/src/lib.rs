use proc_macro::TokenStream;
use quote::quote;
use syn::{
    Data, DeriveInput, Ident, Token,
    parse::{Parse, ParseStream},
    parse_macro_input,
    punctuated::Punctuated,
};

#[proc_macro_derive(Handler, attributes(subscriptions))]
pub fn handler_derive(input: TokenStream) -> TokenStream {
    let ast = parse_macro_input!(input as DeriveInput);
    if !matches!(&ast.data, Data::Struct(_)) {
        let msg = "The #[derive(Handler)] macro can only be used on a `struct`.";
        return syn::Error::new_spanned(&ast.ident, msg)
            .to_compile_error()
            .into();
    };

    let attr = match ast
        .attrs
        .iter()
        .find(|a| a.path().is_ident("subscriptions"))
    {
        Some(attr) => attr,
        None => {
            let msg = "Missing #[subscriptions(...)] attribute. Please list the events to subscribe to, e.g., #[subscriptions(Chat, PlayerJoin)]";
            return syn::Error::new_spanned(&ast.ident, msg)
                .to_compile_error()
                .into();
        }
    };

    let subscriptions = match attr.parse_args::<SubscriptionsListParser>() {
        Ok(list) => list.events,
        Err(e) => {
            return e.to_compile_error().into();
        }
    };

    let subscription_variants = subscriptions.iter().map(|ident| {
        quote! { types::EventType::#ident }
    });

    let struct_name = &ast.ident;

    let output = quote! {
        impl dragonfly_plugin::PluginSubscriptions for #struct_name {
            fn get_subscriptions(&self) -> Vec<types::EventType> {
                vec![
                    #( #subscription_variants ),*
                ]
            }
        }
    };

    output.into()
}

struct SubscriptionsListParser {
    events: Vec<Ident>,
}

impl Parse for SubscriptionsListParser {
    fn parse(input: ParseStream) -> syn::Result<Self> {
        let punctuated_list: Punctuated<syn::Ident, Token![,]> =
            Punctuated::parse_terminated(input)?;

        let events = punctuated_list.into_iter().collect();

        Ok(Self { events })
    }
}
